/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	datastorev1alpha1 "nicolaferraro.me/datastore-crd/pkg/apis/datastore/v1alpha1"
	clientset "nicolaferraro.me/datastore-crd/pkg/client/clientset/versioned"
	datastorescheme "nicolaferraro.me/datastore-crd/pkg/client/clientset/versioned/scheme"
	informers "nicolaferraro.me/datastore-crd/pkg/client/informers/externalversions"
	listers "nicolaferraro.me/datastore-crd/pkg/client/listers/datastore/v1alpha1"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"strings"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/labels"
	"strconv"
	"bufio"
)

const controllerAgentName = "datastore-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a DataStore is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a DataStore fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by DataStore"
	// MessageResourceSynced is the message used for an Event fired when a DataStore
	// is synced successfully
	MessageResourceSynced = "DataStore synced successfully"
)

// Controller is the controller implementation for DataStore resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// datastoreclientset is a clientset for our own API group
	datastoreclientset clientset.Interface

	statefulsetsLister appslisters.StatefulSetLister
	statefulsetsSynced cache.InformerSynced
	podsLister corelisters.PodLister
	podsSynced cache.InformerSynced
	datastoresLister   listers.DataStoreLister
	datastoresSynced   cache.InformerSynced

	workqueue WorkQueue

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new sample controller
func NewController(
	kubeclientset kubernetes.Interface,
	datastoreclientset clientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	datastoreInformerFactory informers.SharedInformerFactory) *Controller {

	// obtain references to shared index informers for the Deployment and DataStore
	// types.
	statefulsetInformer := kubeInformerFactory.Apps().V1().StatefulSets()
	podInformer := kubeInformerFactory.Core().V1().Pods()
	datastoreInformer := datastoreInformerFactory.Datastore().V1alpha1().DataStores()


	// Create event broadcaster
	// Add datastore-crd types to the default Kubernetes Scheme so Events can be
	// logged for datastore-crd types.
	datastorescheme.AddToScheme(scheme.Scheme)
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:      kubeclientset,
		datastoreclientset: datastoreclientset,
		statefulsetsLister:  statefulsetInformer.Lister(),
		statefulsetsSynced: statefulsetInformer.Informer().HasSynced,
		podsLister: podInformer.Lister(),
		podsSynced: podInformer.Informer().HasSynced,
		datastoresLister:   datastoreInformer.Lister(),
		datastoresSynced:   datastoreInformer.Informer().HasSynced,
		workqueue:          NewWorkQueue(),
		recorder:           recorder,
	}

	glog.Info("Setting up event handlers")
	// Set up an event handler for when Datastore resources change
	datastoreInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueDatastore,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueDatastore(new)
		},
	})
	// Set up an event handler for when StatefulSet resources change. This
	// handler will lookup the owner of the given Deployment, and if it is
	// owned by a Datastore resource will enqueue that Datastore resource for
	// processing. This way, we don't need to implement custom logic for
	// handling StatefulSet resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	statefulsetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*appsv1.StatefulSet)
			oldDepl := old.(*appsv1.StatefulSet)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			if object, ok := obj.(*corev1.Pod); ok {
				controller.enqueuePod(object)
			}
		},
	})

	return controller
}

func (c *Controller) updateDeletedPodStatus(pod *corev1.Pod) error {

	if statefulsetRef := metav1.GetControllerOf(pod); statefulsetRef != nil {
		if statefulset, err := c.statefulsetsLister.StatefulSets(pod.Namespace).Get(statefulsetRef.Name); err == nil {
			if datastoreRef := metav1.GetControllerOf(statefulset); datastoreRef != nil {
				if datastore, err := c.datastoresLister.DataStores(pod.Namespace).Get(datastoreRef.Name); err == nil {
					for _, status := range pod.Status.ContainerStatuses {
						if terminated := status.State.Terminated; terminated != nil {
							msgbytes, _, err := bufio.NewReader(strings.NewReader(terminated.Message)).ReadLine()
							var message string
							if err == nil {
								message = string(msgbytes)
							} else {
								message = "<empty>"
							}
							if message == "DECOMMISSIONED" {
								glog.V(4).Infof("Saving pod %s decommission status (true) in datastore %s", pod.Name, datastore.Name)
								datastore = datastore.DeepCopy()
								datastore.Status.SetPodTerminationStatus(pod.Name, true)
								if _, err := c.datastoreclientset.DatastoreV1alpha1().DataStores(datastore.Namespace).Update(datastore); err != nil {
									glog.Warningf("Cannot save pod decommission status (true) in datastore %s", datastore.Name)
								}

								return nil
							} else {
								glog.Warningf("Pod %s container %s termination message is '%s'. Not decommissioned.", pod.Name, status.Name, message)
							}
						}
					}

					glog.V(4).Infof("Saving pod %s decommission status (false) in datastore %s", pod.Name, datastore.Name)
					datastore = datastore.DeepCopy()
					datastore.Status.SetPodTerminationStatus(pod.Name, false)
					if _, err := c.datastoreclientset.DatastoreV1alpha1().DataStores(datastore.Namespace).Update(datastore); err != nil {
						glog.Warningf("Cannot save pod decommission status (false) in datastore %s", datastore.Name)
					}
				}
			}
		}
	}
	return nil
}

func (c *Controller) SignalDecommissionToRequiredPods(datastore *datastorev1alpha1.DataStore) error {
	pods, err := c.podsLister.Pods(datastore.Namespace).List(labels.Everything())
	if err != nil {
		return err
	}

	glog.V(4).Infof("Found %d pods in the namespace", len(pods))

	if datastore.Status.FinalReplicas != nil {
		replicas := int(*datastore.Status.FinalReplicas)
		for _, pod := range pods {
			glog.V(4).Infof("Analyzing pod %s", pod.Name)
			if owner := metav1.GetControllerOf(pod); owner != nil {
				glog.V(4).Infof("Found pod %s with owner %s == %s and owner type %s == %s", pod.Name, owner.Name, datastore.Name, owner.Kind, "StatefulSet")
				if owner.Name == datastore.Name && owner.Kind == "StatefulSet" {
					glog.V(4).Infof("Pod %s is owned", pod.Name)
					num, err := getInstanceNumberString(pod, datastore.Name)
					if err != nil {
						glog.Warningf("Pod %s has a wrong instance number", pod.Name, err)
						continue
					}
					glog.V(4).Infof("Pod %s has instance number %d, target replicas is %d", pod.Name, num, replicas)
					if (num >= replicas) {
						err = c.SignalDecommission(pod.Namespace, pod.Name)
						if err != nil {
							glog.Warningf("An error occurred while signalling the decommission to pod %s: %s", pod.Name, err)
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

func getInstanceNumber(p *corev1.Pod, s *appsv1.StatefulSet) (int, error) {
	return getInstanceNumberString(p, s.Name)
}

func getInstanceNumberString(p *corev1.Pod, s string) (int, error) {
	numStr := strings.TrimPrefix(p.Name, s + "-")
	return strconv.Atoi(numStr)
}

func (c *Controller) SignalDecommission(namespace string, podName string) error {
	glog.Infof("Signalling decommission to pod %s", podName)
	return c.ExecuteCommand(namespace, podName, []string{"touch", "/tmp/decommission-requested"})
}

func (c *Controller) ExecuteCommand(namespace string, podName string, command []string) error {

	pod, err := c.podsLister.Pods(namespace).Get(podName)
	if err != nil {
		return err
	}

	for _, container := range pod.Spec.Containers {

		containerName := container.Name
		req := c.kubeclientset.CoreV1().RESTClient().Post().
			Namespace(namespace).
			Resource("pods").
			Name(podName).
			SubResource("exec").
			Param("container", containerName)

		req.VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command: command,
			Stdin: true,
			Stdout: true,
			Stderr: true,
			TTY: false,
		}, scheme.ParameterCodec)

		restConfig, err := util.NewFactory(nil).ClientConfig()
		if err != nil {
			return err
		}

		exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", req.URL())
		if err != nil {
			return err
		}

		err = exec.Stream(remotecommand.StreamOptions{
			Stdin: strings.NewReader(""),
			Stdout: ioutil.Discard,
			Stderr: ioutil.Discard,
			Tty: false,
		})

		if err != nil {
			return err
		}
	}

	return nil
}



// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	glog.Info("Starting DataStore controller")

	// Wait for the caches to be synced before starting workers
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.podsSynced, c.statefulsetsSynced, c.datastoresSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	glog.Info("Starting workers")
	// Launch two workers to process DataStore resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	glog.Info("Started workers")
	<-stopCh
	glog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, ok := c.workqueue.Dequeue()

	if !ok {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		var key string
		var pod *corev1.Pod
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			if pod, ok = obj.(*corev1.Pod); !ok {

				runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
				return nil
			}
		}
		if pod == nil {
			// Run the syncHandler, passing it the namespace/name string of the
			// DataStore resource to be synced.
			if err := c.syncHandler(key); err != nil {
				return fmt.Errorf("error syncing '%s': %s", key, err.Error())
			}
			glog.V(4).Infof("Successfully synced '%s'", key)
			return nil
		} else {
			// Run the handlePod termination
			if err := c.updateDeletedPodStatus(pod); err != nil {
				return fmt.Errorf("error processing pod '%s': %s", pod, err.Error())
			}
			glog.V(4).Infof("Successfully processed pod '%s'", pod)
			return nil
		}
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the DataStore resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	glog.V(4).Infof("Sync called for key %s", key)

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the DataStore resource with this namespace/name
	datastore, err := c.datastoresLister.DataStores(namespace).Get(name)
	if err != nil {
		// The DataStore resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("datastore '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	statefulsetName := datastore.GetObjectMeta().GetName()

	// Get the deployment with the name specified in DataStore.spec
	statefulset, err := c.statefulsetsLister.StatefulSets(datastore.Namespace).Get(statefulsetName)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		statefulset, err = c.kubeclientset.AppsV1().StatefulSets(datastore.Namespace).Create(newStatefulset(datastore))
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// If the Deployment is not controlled by this DataStore resource, we should log
	// a warning to the event recorder and ret
	if !metav1.IsControlledBy(statefulset, datastore) {
		msg := fmt.Sprintf(MessageResourceExists, statefulset.Name)
		c.recorder.Event(datastore, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf(msg)
	}

	if !datastore.Status.ScalingDown && !datastore.Status.RestartScaling {
		if datastore.Spec.Replicas != nil && *datastore.Spec.Replicas > *statefulset.Spec.Replicas {
			glog.V(4).Infof("DataStore %s replicas: %d, statefulset replicas: %d", name, *datastore.Spec.Replicas, *statefulset.Spec.Replicas)
			statefulset, err = c.updateStatefulset(datastore.Namespace, newStatefulset(datastore))
			c.recorder.Event(datastore, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
		} else if datastore.Spec.Replicas != nil && *datastore.Spec.Replicas < *statefulset.Spec.Replicas {
			// Start scaling down process
			glog.V(4).Infof("DataStore %s replicas: %d, statefulset replicas: %d", name, *datastore.Spec.Replicas, *statefulset.Spec.Replicas)

			datastore, statefulset, err = c.startScalingDown(datastore, statefulset)
		}
	} else if datastore.Status.ScalingDown {

		if (*datastore.Status.FinalReplicas != *statefulset.Spec.Replicas) {
			glog.Warningf("Statefulset %s status inconsistent. Scaling to %d replicas", statefulset.Name, *datastore.Status.FinalReplicas)
			datastore, statefulset, err = c.startScalingDown(datastore, statefulset)
		} else {
			// StatefulSet has been already requested to scale down
			if *datastore.Status.Replicas == *datastore.Status.FinalReplicas {
				// reached the desired point, checking if pod is terminated
				lastPodName := statefulset.Name + "-" + strconv.Itoa(int((*datastore.Status.FinalReplicas)))
				glog.V(4).Infof("Wating for pod %s to be deleted", lastPodName)
				if lastPod, _ := c.podsLister.Pods(datastore.Namespace).Get(lastPodName); lastPod == nil {
					decommissioned := true
					for i := int(*datastore.Status.FinalReplicas); i<int(*datastore.Status.InitialReplicas); i++ {
						podName := statefulset.Name + "-" + strconv.Itoa(i)
						if termStatus := datastore.Status.GetPodTerminationStatus(podName); termStatus != nil {
							decommissioned = decommissioned && termStatus.Decommissioned
						} else {
							decommissioned = false
						}
					}

					if (decommissioned) {
						datastore, err = c.updateDatastore(datastore, func(s *datastorev1alpha1.DataStore) {
							for i := int(*datastore.Status.FinalReplicas); i<int(*datastore.Status.InitialReplicas); i++ {
								podName := statefulset.Name + "-" + strconv.Itoa(i)
								s.Status.RemovePodTerminationStatus(podName)
							}
							s.Status.PodTerminationStatuses = []datastorev1alpha1.DataStorePodTerminationStatus{}
							s.Status.ScalingDown = false
							s.Status.RestartScaling = false
							s.Status.Replicas = &statefulset.Status.Replicas
							s.Status.ReadyReplicas = &statefulset.Status.ReadyReplicas
							s.Status.InitialReplicas = datastore.Status.FinalReplicas
						})
						if err != nil {
							glog.Warningf("Unable to finish scaling down the DataStore %s: %s", datastore.Name, err)
							return err
						}
						c.recorder.Event(datastore, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
					} else {
						datastore, statefulset, err = c.restartScaling(datastore, statefulset)
					}
				}
			}
		}

	} else if datastore.Status.RestartScaling {

		if (*datastore.Status.InitialReplicas != *statefulset.Spec.Replicas) {
			glog.Warningf("Statefulset %s status inconsistent. Scaling to %d replicas", statefulset.Name, datastore.Status.InitialReplicas)
			datastore, statefulset, err = c.restartScaling(datastore, statefulset)
		} else {
			if *datastore.Status.ReadyReplicas == *datastore.Status.InitialReplicas {
				glog.Infof("Statefulset %s status reverted. Now scaling down again to %d replicas", statefulset.Name, datastore.Status.FinalReplicas)
				datastore, statefulset, err = c.startScalingDown(datastore, statefulset)
			}
		}
	}


	// If an error occurs during Update, we'll requeue the item so we can
	// attempt processing again later. THis could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	if datastore.Status.Replicas == nil || datastore.Status.ReadyReplicas == nil || *datastore.Status.Replicas != statefulset.Status.Replicas || *datastore.Status.ReadyReplicas != statefulset.Status.ReadyReplicas {
		glog.V(4).Infof("StatefulSet %s status not synchronized", statefulset.Name)
		datastore, err = c.updateDatastore(datastore, func(s *datastorev1alpha1.DataStore) {
			s.Status.Replicas = &statefulset.Status.Replicas
			s.Status.ReadyReplicas = &statefulset.Status.ReadyReplicas
		})
		if err != nil {
			glog.Warningf("Unable to synchronize replicas of DataStore %s: %s", datastore.Name, err)
			return err
		}
	}

	return nil
}

func (c *Controller) startScalingDown(datastore *datastorev1alpha1.DataStore, statefulset *appsv1.StatefulSet) (*datastorev1alpha1.DataStore, *appsv1.StatefulSet, error) {
	err := c.SignalDecommissionToRequiredPods(datastore)
	if err != nil {
		glog.Warning("Unable to signal decommission to all pods")
		return datastore, statefulset, err
	}

	initialReplicas := statefulset.Spec.Replicas
	statefulset, err = c.updateStatefulset(datastore.Namespace, newStatefulset(datastore))
	if err != nil {
		glog.Warningf("Unable to update the stateful set %s: %s", statefulset.Name, err)
		return datastore, statefulset, err
	}

	glog.Infof("StatefulSet %s scaling down to %d replicas", statefulset.Name, *datastore.Spec.Replicas)
	datastore, err = c.updateDatastore(datastore, func(s *datastorev1alpha1.DataStore) {
		s.Status.ScalingDown = true
		s.Status.Replicas = &statefulset.Status.Replicas
		s.Status.ReadyReplicas = &statefulset.Status.ReadyReplicas
		s.Status.InitialReplicas = initialReplicas
		s.Status.FinalReplicas = datastore.Spec.Replicas
	})
	if err != nil {
		glog.Warningf("Unable to update the status of DataStore %s: %s", datastore.Name, err)
		return datastore, statefulset, err
	}
	return datastore, statefulset, err
}

func (c *Controller) restartScaling(datastore *datastorev1alpha1.DataStore, statefulset *appsv1.StatefulSet) (*datastorev1alpha1.DataStore, *appsv1.StatefulSet, error) {
	newStatefulset := newStatefulset(datastore)
	newStatefulset.Spec.Replicas = datastore.Status.InitialReplicas
	statefulset, err := c.updateStatefulset(datastore.Namespace, newStatefulset)
	if err != nil {
		glog.Warningf("Unable to update the stateful set %s: %s", statefulset.Name, err)
		return datastore, statefulset, err
	}

	datastore, err = c.updateDatastore(datastore, func(s *datastorev1alpha1.DataStore) {
		s.Status.PodTerminationStatuses = []datastorev1alpha1.DataStorePodTerminationStatus{}
		s.Status.ScalingDown = false
		s.Status.RestartScaling = true
		s.Status.Replicas = &statefulset.Status.Replicas
		s.Status.ReadyReplicas = &statefulset.Status.ReadyReplicas
	})
	if err != nil {
		glog.Warningf("Unable to restart scaling down on DataStore %s: %s", datastore.Name, err)
		return datastore, statefulset, err
	}
	return datastore, statefulset, err
}

// enqueueDataStore takes a Datastore resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than DataStore.
func (c *Controller) enqueueDatastore(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.Enqueue(key)
}

func (c *Controller) updateStatefulset(namespace string, statefulset *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	// avoid api rate limiting
	//time.Sleep(200)
	return c.kubeclientset.AppsV1().StatefulSets(namespace).Update(statefulset)
}

func (c *Controller) updateDatastore(datastore *datastorev1alpha1.DataStore, change func(store *datastorev1alpha1.DataStore)) (*datastorev1alpha1.DataStore, error) {
	// avoid api rate limiting
	//time.Sleep(200)

	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	datastoreCopy := datastore.DeepCopy()
	change(datastoreCopy)
	// Until #38113 is merged, we must use Update instead of UpdateStatus to
	// update the Status block of the DataStore resource. UpdateStatus will not
	// allow changes to the Spec of the resource, which is ideal for ensuring
	// nothing other than resource status has been updated.
	return c.datastoreclientset.DatastoreV1alpha1().DataStores(datastore.Namespace).Update(datastoreCopy)
}

// enqueueDataStore takes a Datastore resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than DataStore.
func (c *Controller) enqueuePod(obj *corev1.Pod) {
	c.workqueue.Enqueue(obj)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the DataStore resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that DataStore resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		glog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	glog.V(4).Infof("received object '%s' with name '%s'", object.GetSelfLink(), object.GetName())
	glog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a DataStore, we should not do anything more
		// with it.
		if ownerRef.Kind != "DataStore" {
			return
		}

		datastore, err := c.datastoresLister.DataStores(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			glog.V(4).Infof("ignoring orphaned object '%s' of datastore '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueueDatastore(datastore)
		return
	}
}

// newStatefulset creates a new Deployment for a DataStore resource. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the DataStore resource that 'owns' it.
func newStatefulset(datastore *datastorev1alpha1.DataStore) *appsv1.StatefulSet {

	statefulset := datastore.Spec.StatefulSetSpec.DeepCopy()
	statefulset.Replicas = datastore.Spec.Replicas

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      datastore.ObjectMeta.Name,
			Namespace: datastore.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(datastore, schema.GroupVersionKind{
					Group:   datastorev1alpha1.SchemeGroupVersion.Group,
					Version: datastorev1alpha1.SchemeGroupVersion.Version,
					Kind:    "DataStore",
				}),
			},
		},
		Spec: *statefulset,
	}
}

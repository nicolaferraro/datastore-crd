/*
Copyright 2018 The Kubernetes Authors.

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

package v1alpha1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
	v1alpha1 "nicolaferraro.me/datastore-crd/pkg/apis/datastore/v1alpha1"
	scheme "nicolaferraro.me/datastore-crd/pkg/client/clientset/versioned/scheme"
)

// DataStoresGetter has a method to return a DataStoreInterface.
// A group's client should implement this interface.
type DataStoresGetter interface {
	DataStores(namespace string) DataStoreInterface
}

// DataStoreInterface has methods to work with DataStore resources.
type DataStoreInterface interface {
	Create(*v1alpha1.DataStore) (*v1alpha1.DataStore, error)
	Update(*v1alpha1.DataStore) (*v1alpha1.DataStore, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.DataStore, error)
	List(opts v1.ListOptions) (*v1alpha1.DataStoreList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.DataStore, err error)
	DataStoreExpansion
}

// dataStores implements DataStoreInterface
type dataStores struct {
	client rest.Interface
	ns     string
}

// newDataStores returns a DataStores
func newDataStores(c *DatastoreV1alpha1Client, namespace string) *dataStores {
	return &dataStores{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the dataStore, and returns the corresponding dataStore object, and an error if there is any.
func (c *dataStores) Get(name string, options v1.GetOptions) (result *v1alpha1.DataStore, err error) {
	result = &v1alpha1.DataStore{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("datastores").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of DataStores that match those selectors.
func (c *dataStores) List(opts v1.ListOptions) (result *v1alpha1.DataStoreList, err error) {
	result = &v1alpha1.DataStoreList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("datastores").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested dataStores.
func (c *dataStores) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("datastores").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a dataStore and creates it.  Returns the server's representation of the dataStore, and an error, if there is any.
func (c *dataStores) Create(dataStore *v1alpha1.DataStore) (result *v1alpha1.DataStore, err error) {
	result = &v1alpha1.DataStore{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("datastores").
		Body(dataStore).
		Do().
		Into(result)
	return
}

// Update takes the representation of a dataStore and updates it. Returns the server's representation of the dataStore, and an error, if there is any.
func (c *dataStores) Update(dataStore *v1alpha1.DataStore) (result *v1alpha1.DataStore, err error) {
	result = &v1alpha1.DataStore{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("datastores").
		Name(dataStore.Name).
		Body(dataStore).
		Do().
		Into(result)
	return
}

// Delete takes name of the dataStore and deletes it. Returns an error if one occurs.
func (c *dataStores) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("datastores").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *dataStores) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("datastores").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched dataStore.
func (c *dataStores) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.DataStore, err error) {
	result = &v1alpha1.DataStore{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("datastores").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}

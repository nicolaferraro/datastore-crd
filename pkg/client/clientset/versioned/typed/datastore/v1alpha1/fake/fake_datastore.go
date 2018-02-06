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

package fake

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
	v1alpha1 "nicolaferraro.me/datastore-crd/pkg/apis/datastore/v1alpha1"
)

// FakeDataStores implements DataStoreInterface
type FakeDataStores struct {
	Fake *FakeDatastoreV1alpha1
	ns   string
}

var datastoresResource = schema.GroupVersionResource{Group: "datastore.nicolaferraro.me", Version: "v1alpha1", Resource: "datastores"}

var datastoresKind = schema.GroupVersionKind{Group: "datastore.nicolaferraro.me", Version: "v1alpha1", Kind: "DataStore"}

// Get takes name of the dataStore, and returns the corresponding dataStore object, and an error if there is any.
func (c *FakeDataStores) Get(name string, options v1.GetOptions) (result *v1alpha1.DataStore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(datastoresResource, c.ns, name), &v1alpha1.DataStore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.DataStore), err
}

// List takes label and field selectors, and returns the list of DataStores that match those selectors.
func (c *FakeDataStores) List(opts v1.ListOptions) (result *v1alpha1.DataStoreList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(datastoresResource, datastoresKind, c.ns, opts), &v1alpha1.DataStoreList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.DataStoreList{}
	for _, item := range obj.(*v1alpha1.DataStoreList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested dataStores.
func (c *FakeDataStores) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(datastoresResource, c.ns, opts))

}

// Create takes the representation of a dataStore and creates it.  Returns the server's representation of the dataStore, and an error, if there is any.
func (c *FakeDataStores) Create(dataStore *v1alpha1.DataStore) (result *v1alpha1.DataStore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(datastoresResource, c.ns, dataStore), &v1alpha1.DataStore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.DataStore), err
}

// Update takes the representation of a dataStore and updates it. Returns the server's representation of the dataStore, and an error, if there is any.
func (c *FakeDataStores) Update(dataStore *v1alpha1.DataStore) (result *v1alpha1.DataStore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(datastoresResource, c.ns, dataStore), &v1alpha1.DataStore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.DataStore), err
}

// Delete takes name of the dataStore and deletes it. Returns an error if one occurs.
func (c *FakeDataStores) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(datastoresResource, c.ns, name), &v1alpha1.DataStore{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeDataStores) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(datastoresResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.DataStoreList{})
	return err
}

// Patch applies the patch and returns the patched dataStore.
func (c *FakeDataStores) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.DataStore, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(datastoresResource, c.ns, name, data, subresources...), &v1alpha1.DataStore{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.DataStore), err
}

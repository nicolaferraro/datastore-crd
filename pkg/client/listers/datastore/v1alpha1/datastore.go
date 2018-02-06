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

// This file was automatically generated by lister-gen

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1alpha1 "nicolaferraro.me/datastore-crd/pkg/apis/datastore/v1alpha1"
)

// DataStoreLister helps list DataStores.
type DataStoreLister interface {
	// List lists all DataStores in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.DataStore, err error)
	// DataStores returns an object that can list and get DataStores.
	DataStores(namespace string) DataStoreNamespaceLister
	DataStoreListerExpansion
}

// dataStoreLister implements the DataStoreLister interface.
type dataStoreLister struct {
	indexer cache.Indexer
}

// NewDataStoreLister returns a new DataStoreLister.
func NewDataStoreLister(indexer cache.Indexer) DataStoreLister {
	return &dataStoreLister{indexer: indexer}
}

// List lists all DataStores in the indexer.
func (s *dataStoreLister) List(selector labels.Selector) (ret []*v1alpha1.DataStore, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.DataStore))
	})
	return ret, err
}

// DataStores returns an object that can list and get DataStores.
func (s *dataStoreLister) DataStores(namespace string) DataStoreNamespaceLister {
	return dataStoreNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// DataStoreNamespaceLister helps list and get DataStores.
type DataStoreNamespaceLister interface {
	// List lists all DataStores in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.DataStore, err error)
	// Get retrieves the DataStore from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.DataStore, error)
	DataStoreNamespaceListerExpansion
}

// dataStoreNamespaceLister implements the DataStoreNamespaceLister
// interface.
type dataStoreNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all DataStores in the indexer for a given namespace.
func (s dataStoreNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.DataStore, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.DataStore))
	})
	return ret, err
}

// Get retrieves the DataStore from the indexer for a given namespace and name.
func (s dataStoreNamespaceLister) Get(name string) (*v1alpha1.DataStore, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("datastore"), name)
	}
	return obj.(*v1alpha1.DataStore), nil
}
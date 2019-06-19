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

package cloud_provider

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/cloud-provider-baiducloud/pkg/cloud-sdk/blb"
)

func (bc *Baiducloud) reconcileBackendServers(service *v1.Service, nodes []*v1.Node, lb *blb.LoadBalancer) error {
	// extract annotation
	anno, err := ExtractServiceAnnotation(service)
	if err != nil {
		return fmt.Errorf("failed to ExtractServiceAnnotation %s, err: %v", service.Name, err)
	}
	// default rs num of a blb is 50
	targetRsNum := 50
	if anno.LoadBalancerRsNum > 0 {
		if anno.LoadBalancerRsNum < 100 {
			targetRsNum = anno.LoadBalancerRsNum
		} else {
			glog.Infof("annotation rs num %d > 100, not use this value", anno.LoadBalancerRsNum)
		}
	}
	if len(nodes) < targetRsNum {
		targetRsNum = len(nodes)
	}
	glog.Infof("nodes num is %d, target Rs num is %d", len(nodes), targetRsNum)

	// turn candidate nodes list to map
	candidateBackendsMap := make(map[string]int, 0)
	for _, node := range nodes {
		splitted := strings.Split(node.Spec.ProviderID, "//")
		if len(splitted) < 1 {
			glog.Warningf("node %s has no spec.providerId", node.Name)
			continue
		}
		name := splitted[1]
		candidateBackendsMap[name] = 0
	}

	// get all existing rs from lb and change to map
	allRs, err := bc.getAllBackendServer(lb)
	if err != nil {
		return err
	}
	existingRsMap := make(map[string]int, 0)
	for _, rs := range allRs {
		existingRsMap[rs.InstanceId] = 0
	}

	// find rs to delete
	var nodesToAdd, nodesToDelete []string
	for rs := range existingRsMap {
		if _, exist := candidateBackendsMap[rs]; !exist {
			nodesToDelete = append(nodesToDelete, rs)
		}
	}
	glog.Infof("find nodes %v to delete from BLB %s", nodesToDelete, lb.BlbId)

	// find rs to add
	if len(existingRsMap) < targetRsNum {
		numToAdd := targetRsNum - len(existingRsMap)
		for node := range candidateBackendsMap {
			if numToAdd == 0 {
				break
			}
			if _, exist := existingRsMap[node]; !exist {
				nodesToAdd = append(nodesToAdd, node)
				numToAdd = numToAdd - 1
			}
		}
	}
	glog.Infof("find nodes %v to add to BLB %s", nodesToAdd, lb.BlbId)

	// add rs
	var addList []blb.BackendServer
	if len(nodesToAdd) > 0 {
		for _, insId := range nodesToAdd {
			addList = append(addList, blb.BackendServer{
				InstanceId: insId,
				Weight: 100,
			})
		}
	}
	if len(addList) > 0 {
		args := blb.AddBackendServersArgs{
			LoadBalancerId: lb.BlbId,
			BackendServerList: addList,
		}
		err = bc.clientSet.Blb().AddBackendServers(&args)
		if err != nil {
			return err
		}
	}

	// remove rs
	if len(nodesToDelete) > 0 {
		args := blb.RemoveBackendServersArgs{
			LoadBalancerId: lb.BlbId,
			BackendServerList: nodesToDelete,
		}
		err = bc.clientSet.Blb().RemoveBackendServers(&args)
		if err != nil {
			return err
		}
	}

	return nil
}

func (bc *Baiducloud) getAllBackendServer(lb *blb.LoadBalancer) ([]blb.BackendServer, error) {
	args := blb.DescribeBackendServersArgs{
		LoadBalancerId: lb.BlbId,
	}
	bs, err := bc.clientSet.Blb().DescribeBackendServers(&args)
	if err != nil {
		return nil, err
	}
	return bs, nil
}

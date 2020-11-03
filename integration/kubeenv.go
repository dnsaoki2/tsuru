// Copyright 2020 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package integration

// KubeenvClusterManager represents a kubectl context
type KubeenvClusterManager struct {
	env *Environment
}

func (m *KubeenvClusterManager) Name() string {
	return "kubeenv"
}

func (m *KubeenvClusterManager) Provisioner() string {
	return kubernetesProvisioner
}

func (m *KubeenvClusterManager) Start() *Result {
	return &Result{}
}

func (m *KubeenvClusterManager) Delete() *Result {
	return &Result{}
}

func (m *KubeenvClusterManager) UpdateParams() ([]string, bool) {
	addr := m.env.Get("cluster_addr")
	cacert := m.env.Get("cluster_cacert")
	username := m.env.Get("cluster_username")
	password := m.env.Get("cluster_password")
	token := m.env.Get("cluster_token")

	params := []string{"--addr", addr, "--cacert", cacert}

	if username != "" && password != "" {
		params = append(params, "--custom", "username="+username)
		params = append(params, "--custom", "password="+password)
	} else {
		params = append(params, "--custom", "token="+token)
	}

	return params, false
}

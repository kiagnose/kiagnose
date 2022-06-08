/*
 * This file is part of the kiagnose project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2022 Red Hat, Inc.
 *
 */

package netattachdef

import "github.com/containernetworking/cni/libcni"

func ListCNIPluginTypes(config string) []string {
	if netConfList, err := libcni.ConfListFromBytes([]byte(config)); err == nil {
		var cniPlugins []string
		for _, plugin := range netConfList.Plugins {
			cniPlugins = append(cniPlugins, plugin.Network.Type)
		}
		return cniPlugins
	}

	if netConf, err := libcni.ConfFromBytes([]byte(config)); err == nil {
		return []string{netConf.Network.Type}
	}

	return nil
}

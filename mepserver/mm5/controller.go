/*
 * Copyright 2020 Huawei Technologies Co., Ltd.
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
 */

package mm5

import (
	"net/http"

	"github.com/apache/servicecomb-service-center/pkg/rest"
	v4 "github.com/apache/servicecomb-service-center/server/rest/controller/v4"
	"mepserver/common"
	"mepserver/common/arch/workspace"
	meputil "mepserver/common/util"
	"mepserver/mm5/plans"
)

func init() {
	initMm5Router()
}

func initMm5Router() {
	rest.
		RegisterServant(&Mm5Service{})
}

type Mm5Service struct {
	v4.MicroServiceService
}

func (m *Mm5Service) URLPatterns() []rest.Route {
	return []rest.Route{
		// Platform Capability Query
		{Method: rest.HTTP_METHOD_GET, Path: meputil.CapabilityPath, Func: getPlatformCapabilities},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.CapabilityPath + meputil.CapabilityIdPath, Func: getPlatformCapability},
	}
}

func getPlatformCapabilities(w http.ResponseWriter, r *http.Request) {
	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeCapabilityQueryReq{},
		&plans.CapabilitiesGet{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func getPlatformCapability(w http.ResponseWriter, r *http.Request) {
	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeCapabilityQueryReq{},
		&plans.CapabilityGet{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

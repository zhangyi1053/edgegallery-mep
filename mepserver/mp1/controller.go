/*
 * Copyright 2020-2021 Huawei Technologies Co., Ltd.
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

// Package mp1 implements rest api route controller
package mp1

import (
	"encoding/json"
	"fmt"
	"github.com/apache/servicecomb-service-center/pkg/util"
	"mepserver/common/config"
	"mepserver/common/extif/backend"
	"mepserver/common/extif/dataplane"
	dpCommon "mepserver/common/extif/dataplane/common"
	"mepserver/common/extif/dns"
	"mepserver/common/models"
	"net/http"

	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/rest"
	v4 "github.com/apache/servicecomb-service-center/server/rest/controller/v4"

	"mepserver/common"
	"mepserver/common/arch/workspace"
	meputil "mepserver/common/util"
	"mepserver/mp1/plans"
)

const transportNameRest = "REST"

type APIHookFunc func() models.EndPointInfo

type APIGwHook struct {
	APIHook APIHookFunc
}

var apihook APIGwHook

// SetAPIHook set api gw hook
func SetAPIHook(hook APIGwHook) {
	apihook = hook
}

func init() {
	initRouter()
}

func initRouter() {
	mp1 := &Mp1Service{}
	if err := mp1.Init(); err != nil {
		log.Errorf(err, "Mp1 interface initialization failed.")
		//os.Exit(1) # Init function cannot be mocked by test. Hence removed this.
	}
	rest.RegisterServant(mp1)
}

// Mp1Service represents the mp1 service object
type Mp1Service struct {
	v4.MicroServiceService
	config    *config.MepServerConfig
	dnsAgent  dns.DNSAgent
	dataPlane dataplane.DataPlane
}

// Init initialize mp1 service
func (m *Mp1Service) Init() error {
	mepConfig, err := config.LoadMepServerConfig()
	if err != nil {
		return fmt.Errorf("error: reading configuration failed")
	}
	m.config = mepConfig

	// Checking if local or both is configured
	var dnsAgent dns.DNSAgent
	if m.config.DNSAgent.Type != meputil.DnsAgentTypeDataPlane {
		dnsAgent = dns.NewRestDNSAgent(mepConfig)
	}
	m.dnsAgent = dnsAgent
	// select data plane as per configuration
	dataPlane := dpCommon.CreateDataPlane(mepConfig)
	if dataPlane == nil {
		return fmt.Errorf("error: unsupported data-plane")
	}

	if err := dataPlane.InitDataPlane(mepConfig); err != nil {
		return err
	}
	m.dataPlane = dataPlane
	log.Infof("Data plane initialized to %s.", m.config.DataPlane.Type)

	if err := m.InitTransportInfo(); err != nil {
		//return err
	}

	return nil
}

func (m *Mp1Service) fillTransportInfo(tpInfos *models.TransportInfo) {
	tpInfos.ID = util.GenerateUuid()
	tpInfos.Name = "REST"
	tpInfos.Description = "REST API"
	tpInfos.TransType = "REST_HTTP"
	tpInfos.Protocol = "HTTP"
	tpInfos.Version = "2.0"
	var theArray = make([]string, 1)
	theArray[0] = "OAUTH2_CLIENT_CREDENTIALS"
	tpInfos.Security.OAuth2Info.GrantTypes = theArray
	tpInfos.Security.OAuth2Info.TokenEndpoint = "/mep/token"
}

func (m *Mp1Service) checkTransportIsExists(tpInfos *models.TransportInfo) bool {
	respLists, err := backend.GetRecords(meputil.TransportInfoPath)
	if err != 0 {
		log.Errorf(nil, "Get transport info from data-store failed.")
		return false
	}

	for _, value := range respLists {
		var transportInfo *models.TransportInfo
		err := json.Unmarshal(value, &transportInfo)
		if err != nil {
			log.Errorf(nil, "Transport Info decode failed.")
			return false
		}

		if transportInfo.Name == transportNameRest {
			log.Infof("Transport info exists for  %v", transportInfo.Name)
			return true
		}
	}
	return false
}

func (m *Mp1Service) InitTransportInfo() error {
	var transportInfos models.TransportInfo
	m.fillTransportInfo(&transportInfos)

	if m.checkTransportIsExists(&transportInfos) == true {
		return nil
	}

	updateJSON, err := json.Marshal(transportInfos)
	if err != nil {
		log.Errorf(err, "Can not marshal the input transport info.")
		return fmt.Errorf("error: Can not marshal the input transport info")
	}

	resultErr := backend.PutRecord(meputil.TransportInfoPath+transportInfos.ID, updateJSON)
	if resultErr != 0 {
		log.Errorf(nil, "Transport info update on etcd failed.")
		return fmt.Errorf("error: Transport info update on etcd failed")
	}

	log.Infof("Transport info added for %s.", transportInfos.Name)
	return nil
}

// URLPatterns handles url mappings
func (m *Mp1Service) URLPatterns() []rest.Route {
	return []rest.Route{
		// appSubscriptions
		{Method: rest.HTTP_METHOD_POST, Path: meputil.AppSubscribePath, Func: m.doAppSubscribe},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.AppSubscribePath, Func: m.getAppSubscribes},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.AppSubscribePath + meputil.SubscriptionIdPath,
			Func: m.getOneAppSubscribe},
		{Method: rest.HTTP_METHOD_DELETE, Path: meputil.AppSubscribePath + meputil.SubscriptionIdPath,
			Func: m.delOneAppSubscribe},
		// appServices
		{Method: rest.HTTP_METHOD_POST, Path: meputil.AppServicesPath, Func: m.serviceRegister},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.AppServicesPath, Func: m.serviceDiscover},
		{Method: rest.HTTP_METHOD_PUT, Path: meputil.AppServicesPath + meputil.ServiceIdPath, Func: m.serviceUpdate},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.AppServicesPath + meputil.ServiceIdPath, Func: m.getOneService},
		{Method: rest.HTTP_METHOD_DELETE, Path: meputil.AppServicesPath + meputil.ServiceIdPath, Func: m.serviceDelete},
		// MEC Application Support API - appSubscriptions
		{Method: rest.HTTP_METHOD_POST, Path: meputil.EndAppSubscribePath, Func: m.appEndSubscribe},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.EndAppSubscribePath, Func: m.getAppEndSubscribes},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.EndAppSubscribePath + meputil.SubscriptionIdPath,
			Func: m.getEndAppOneSubscribe},
		{Method: rest.HTTP_METHOD_DELETE, Path: meputil.EndAppSubscribePath + meputil.SubscriptionIdPath,
			Func: m.delEndAppOneSubscribe},
		// DNS
		{Method: rest.HTTP_METHOD_GET, Path: meputil.DNSRulesPath, Func: m.getDnsRules},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.DNSRulesPath + meputil.DNSRuleIdPath, Func: m.getDnsRule},
		{Method: rest.HTTP_METHOD_PUT, Path: meputil.DNSRulesPath + meputil.DNSRuleIdPath, Func: m.dnsRuleUpdate},
		// HeartBeat
		{Method: rest.HTTP_METHOD_GET, Path: meputil.AppServicesPath + meputil.ServiceIdPath + meputil.Liveness,
			Func: m.getHeartbeat},
		{Method: rest.HTTP_METHOD_PUT, Path: meputil.AppServicesPath + meputil.ServiceIdPath + meputil.Liveness,
			Func: m.heartbeatService},
		//Liveness and readiness
		{Method: rest.HTTP_METHOD_GET, Path: "/health", Func: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("ok"))
		}},
		// services
		{Method: rest.HTTP_METHOD_GET, Path: meputil.ServicesPath, Func: m.serviceDiscover},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.ServicesPath + "/:serviceId", Func: m.getOneService},
		//traffic Rules
		{Method: rest.HTTP_METHOD_GET, Path: meputil.TrafficRulesPath, Func: m.getTrafficRules},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.TrafficRulesPath + meputil.TrafficRuleIdPath, Func: m.getTrafficRule},
		{Method: rest.HTTP_METHOD_PUT, Path: meputil.TrafficRulesPath + meputil.TrafficRuleIdPath, Func: m.trafficRuleUpdate},
		//NTP
		{Method: rest.HTTP_METHOD_GET, Path: meputil.TimingPath + meputil.CurrentTIme, Func: m.getCurrentTime},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.TimingPath + meputil.TimingCaps, Func: m.getTimingCaps},
		{Method: rest.HTTP_METHOD_GET, Path: meputil.TransportPath, Func: m.getTransports},
	}
}

func (m *Mp1Service) appEndSubscribe(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try((&plans.DecodeRestReq{}).WithBody(&models.AppTerminationNotificationSubscription{}),
		(&plans.AppSubscribeLimit{}).WithType(meputil.AppTerminationNotificationSubscription),
		(&plans.SubscribeIst{}).WithType(meputil.AppTerminationNotificationSubscription))
	workPlan.Finally(&common.SendHttpRsp{StatusCode: http.StatusCreated})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getAppEndSubscribes(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		(&plans.GetSubscribes{}).WithType(meputil.AppTerminationNotificationSubscription))
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getEndAppOneSubscribe(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		(&plans.GetOneSubscribe{}).WithType(meputil.AppTerminationNotificationSubscription))
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) delEndAppOneSubscribe(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		(&plans.DelOneSubscribe{}).WithType(meputil.AppTerminationNotificationSubscription))
	workPlan.Finally(&common.SendHttpRsp{StatusCode: http.StatusNoContent})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) doAppSubscribe(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		(&plans.DecodeRestReq{}).WithBody(&models.SerAvailabilityNotificationSubscription{}),
		(&plans.AppSubscribeLimit{}).WithType(meputil.SerAvailabilityNotificationSubscription),
		(&plans.SubscribeIst{}).WithType(meputil.SerAvailabilityNotificationSubscription))
	workPlan.Finally(&common.SendHttpRsp{StatusCode: http.StatusCreated})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getAppSubscribes(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		(&plans.GetSubscribes{}).WithType(meputil.SerAvailabilityNotificationSubscription))
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getOneAppSubscribe(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		(&plans.GetOneSubscribe{}).WithType(meputil.SerAvailabilityNotificationSubscription))
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) delOneAppSubscribe(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		(&plans.DelOneSubscribe{}).WithType(meputil.SerAvailabilityNotificationSubscription))
	workPlan.Finally(&common.SendHttpRsp{StatusCode: http.StatusNoContent})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) serviceRegister(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		(&plans.DecodeRestReq{}).WithBody(&models.ServiceInfo{}),
		&plans.RegisterLimit{},
		&plans.RegisterServiceId{},
		&plans.RegisterServiceInst{})
	workPlan.Finally(&common.SendHttpRsp{StatusCode: http.StatusCreated})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) serviceDiscover(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&DiscoverDecode{},
		&DiscoverService{},
		&ToStrDiscover{},
		&RspHook{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) serviceUpdate(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		(&plans.DecodeRestReq{}).WithBody(&models.ServiceInfo{}),
		&plans.UpdateInstance{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getOneService(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.GetOneDecode{},
		&plans.GetOneInstance{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)

}

func (m *Mp1Service) serviceDelete(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeRestReq{},
		&plans.DeleteService{})
	workPlan.Finally(&common.SendHttpRsp{StatusCode: http.StatusNoContent})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getDnsRules(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeDnsRestReq{},
		&plans.DNSRulesGet{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getDnsRule(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeDnsRestReq{},
		&plans.DNSRuleGet{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) dnsRuleUpdate(w http.ResponseWriter, r *http.Request) {
	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		(&plans.DecodeDnsRestReq{}).WithBody(&dataplane.DNSRule{}),
		(&plans.DNSRuleUpdate{}).WithDNSAgent(m.dnsAgent).WithDataPlane(m.dataPlane))
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getHeartbeat(w http.ResponseWriter, r *http.Request) {
	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.GetOneDecodeHeartbeat{},
		&plans.GetOneInstanceHeartbeat{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) heartbeatService(w http.ResponseWriter, r *http.Request) {
	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		(&plans.DecodeHeartbeatRestReq{}).WithBodies(&models.ServiceLivenessUpdate{}),
		&plans.UpdateHeartbeat{})
	workPlan.Finally(&common.SendHttpRsp{StatusCode: http.StatusNoContent})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getTrafficRules(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeTrafficRestReq{},
		&plans.TrafficRulesGet{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getTrafficRule(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		&plans.DecodeTrafficRestReq{},
		&plans.TrafficRuleGet{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) trafficRuleUpdate(w http.ResponseWriter, r *http.Request) {
	workPlan := NewWorkSpace(w, r)
	workPlan.Try(
		(&plans.DecodeTrafficRestReq{}).WithBody(&dataplane.TrafficRule{}),
		(&plans.TrafficRuleUpdate{}).WithDataPlane(m.dataPlane))
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getCurrentTime(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(&plans.CurrentTimeGet{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getTimingCaps(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(&plans.TimingCaps{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

func (m *Mp1Service) getTransports(w http.ResponseWriter, r *http.Request) {

	workPlan := NewWorkSpace(w, r)
	workPlan.Try(&plans.Transports{})
	workPlan.Finally(&common.SendHttpRsp{})

	workspace.WkRun(workPlan)
}

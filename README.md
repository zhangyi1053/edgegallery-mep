# EdgeGallery MEP project

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

## Introduction

Edgegallery MEP is an open source implementation of MEC platform according to ETSI MEC 003 [1] and 011 [2]
documentation.

The MEC platform, as defined in ETSI GS MEC 003, offers an environment where MEC applications may discover, advertise,
consume and offer MEC services. Upon receipt of update, activation or deactivation of traffic rules from the MEC
platform manager, applications or services, the MEC platform instructs the data plane accordingly. The MEC platform also
receives DNS records from the MEC platform manager and uses them to configure a DNS proxy/server.

Via Mp1 reference point between the MEC platform and the MEC applications, as defined in ETSI GS MEC 011, the basic
functions are enabled, such as:

* MEC service assistance:
    - authentication and authorization of producing and consuming MEC services;
    - a means for service producing MEC applications to register/deregister towards the MEC platform the MEC services
      they provide, and to update the MEC platform about changes of the MEC service availability;
    - a means to notify the changes of the MEC service availability to the relevant MEC application;
    - discovery of available MEC services;
* MEC application assistance:
    - MEC application availability subscription;
    - MEC application termination subscription;

## MEP architecture

MEP Mp1 service registry and discovery bases on servicecomb service center [3]. Servicecomb service center is a Restful
based service-registry that provides micro-services discovery and micro-service management. MEP utilize its registry
abilities and plugin mechanism to implement Mp1 interfaces. The mep-server module is the core implementations of MEP
server for Mp1 APIs. The APIs is provided for MEC Apps to register or discover services in MEC platform.

## MEP code directory

```
├── kong-plugin
│   ├── appid-header
│   └── kong.conf
├── mepauth
├── mepserver
└── README.md

```

Above is the directory tree of MEP project, their usage is as belows:

- kong-plugin: mep api gateway kong plugin
- mepserver: mep server implementation
- mepauth: mepauth module provide token apply api for Apps

## MEP build & run

Most of the MEP project codes are implemented in golang, the kong plugin in lua. MEP project is released via docker
image.

### Build mep-auth

```
cd mepauth
sudo ./docker-build.sh

```

### Build mep-server

```
cd mepserver
sudo ./docker-build.sh
```

### Run mepauth

```
docker run -itd --name mepauth \
             --cap-drop All \
             --network mep-net \
             --link kong-service:kong-service \
             -v ${MEP_CERTS_DIR}/jwt_publickey:${MEPAUTH_KEYS_DIR}/jwt_publickey:ro \
             -v ${MEP_CERTS_DIR}/jwt_encrypted_privatekey:${MEPAUTH_KEYS_DIR}/jwt_encrypted_privatekey:ro \
             -v ${MEP_CERTS_DIR}/mepserver_tls.crt:${MEPAUTH_SSL_DIR}/server.crt:ro \
             -v ${MEP_CERTS_DIR}/mepserver_tls.key:${MEPAUTH_SSL_DIR}/server.key:ro \
             -v ${MEP_CERTS_DIR}/ca.crt:${MEPAUTH_SSL_DIR}/ca.crt:ro \
             -v ${MEPAUTH_CONF_PATH}:/usr/mep/mprop/mepauth.properties \
             -e "MEPAUTH_APIGW_HOST=kong-service" \
             -e "MEPAUTH_APIGW_PORT=8444"  \
             -e "MEPAUTH_CERT_DOMAIN_NAME=${DOMAIN_NAME}" \
             -e "MEPSERVER_HOST=mepserver" \
             edgegallery/mepauth:latest
```

MEP_CERTS_DIR is where you put mepauth server certificates and keys. MEPAUTH_CONF_PATH is a config file for mepauth.

### Run mepserver

MEP_CERTS_DIR is where you put mep server certificates and keys. MEP_ROOT_KEY_COMPONENT is the root key's random
component of length 256. CERT_PASSPHRASE is the passphrase used to create the certificate.

```
docker run -itd --name mepserver --network mep-net -e "SSL_ROOT=${MEPSERVER_SSL_DIR}" \
                                 --cap-drop All \
                                 -v ${MEP_CERTS_DIR}/mepserver_tls.crt:${MEPSERVER_SSL_DIR}/server.cer:ro \
                                 -v ${MEP_CERTS_DIR}/mepserver_encryptedtls.key:${MEPSERVER_SSL_DIR}/server_key.pem:ro \
                                 -v ${MEP_CERTS_DIR}/ca.crt:${MEPSERVER_SSL_DIR}/trust.cer:ro \
                                 -v ${MEP_CERTS_DIR}/mepserver_cert_pwd:${MEPSERVER_SSL_DIR}/cert_pwd:ro \
                                 -e "ROOT_KEY=${MEP_ROOT_KEY_COMPONENT}" \
                                 -e "TLS_KEY=${CERT_PASSPHRASE}" \
                                 edgegallery/mep:latest
```

More details about the building and installation of MEP, please refer
to [HERE](https://gitee.com/edgegallery/docs/blob/master/Projects/MEP/EdgeGallery%E6%9C%AC%E5%9C%B0%E5%BC%80%E5%8F%91%E9%AA%8C%E8%AF%81%E6%9C%8D%E5%8A%A1%E8%AF%B4%E6%98%8E%E4%B9%A6.md#EG-LDVS-MEP%E9%83%A8%E7%BD%B2%E6%8C%87%E5%AF%BC)
.

## UPF Standard Interface

MEP supports passing the DNS and traffic rules to the data-plane/UPF over the UPF standard interface. As the interface
is not a standardized one and there can be difference in implementation, mepserver has provided an interface file which
anybody can implement as per their requirements and needs.

The interface is available
at [dataplane_if.go](https://gitee.com/edgegallery/mep/blob/master/mepserver/common/extif/dataplane/dataplane_if.go)
location, and a sample implementation
on [none](https://gitee.com/edgegallery/mep/blob/master/mepserver/common/extif/dataplane/none) directory. More details
given below on how to create a new UPF interface.

1. Add a new folder, in parallel to **none** folder and name it as per your convenience.
   ```shell
   ~/mep/mepserver/common/extif/dataplane$ mkdir abcupf
   ~/mep/mepserver/common/extif/dataplane$ ls -l
   total 20
   drwxrwxr-x 2 root1 root1 4096 Jul 27 16:31 abcupf
   drwxrwxr-x 2 root1 root1 4096 Jul 27 16:02 common
   -rw-rw-r-- 1 root1 root1 5077 Jul 27 16:02 dataplane_if.go
   drwxrwxr-x 2 root1 root1 4096 Jul 27 16:02 none
   ~/mep/mepserver/common/extif/dataplane$
   ```
2. Create a new golang file in it and implement all the interfaces defined in the dataplane_if.go file.
    1. **InitDataPlane**: This interface will be called on startup of the mepserver to initialize the data-plane/upf.
       Operations like reading configurations, creating/initializing client, generating tokens etc can be performed in
       this.
    2. **AddTrafficRule**: Will be triggered when mep server receives a traffic rule create request on Mm5 interface or
       a state change request on Mp1.
    3. **SetTrafficRule**: Update a traffic rule instance, triggered on change in Mm5 or Mp1.
    4. **DeleteTrafficRule**: Deletes a traffic rule instance, triggered on update or delete on Mm5 or update request in
       Mp1.
    5. **AddDNSRule**: Creates a DNS rule in data-plane/upf. Triggered on create or modify request on Mm5 or state
       change request in Mp1.
    6. **SetDNSRule**: Updates DNS rule on Mm5 modify request.
    7. **DeleteDNSRule**: Deletes a DNS rule on update or delete request in Mm5 or state change in Mp1.
3. Once the interface file is ready, now it's time update the factory method. For this, we have to update the
   CreateDataPlane function in mep/mepserver/common/extif/dataplane/common/dataplane.go file.
   ```go
   func CreateDataPlane(config *config.MepServerConfig) dataplane.DataPlane {
       if config.DataPlane.Type == meputil.DataPlaneNone {
           return &none.NoneDataPlane{}
       } else if config.DataPlane.Type == "abcUpf" {
           return &abcupf.AbcDataPlane{}
       }
       return nil
   }
   ```
4. After modification, you can build the mepserver container as per the steps given above and replace in the
   Edge-Gallery deployments.
5. We have to change the configuration file as well to reflect the complete modification. For this we need to make the
   below modification in the config.yaml file.
   ```yaml
   dataplane:
     type: abcUpf # same as the one used in CreateDataPlane function
   ```
   You can update this in a running environment using the below command, and then restart the mepserver container to
   reflect new changes.
   ```shell
   $ kubectl -n mep edit secrets mep-config 
   # then replace the data: config.yaml section with updated config file.
   ```

## Reference

[1] https://www.etsi.org/deliver/etsi_gs/MEC/001_099/003/02.01.01_60/gs_MEC003v020101p.pdf

[2] https://www.etsi.org/deliver/etsi_gs/MEC/001_099/011/02.01.01_60/gs_MEC011v020101p.pdf

[3] https://github.com/apache/servicecomb-service-center

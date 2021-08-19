// Copyright © 2021 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fabric

import (
	"fmt"
	"path"

	"github.com/hyperledger-labs/firefly-cli/internal/constants"
	"github.com/hyperledger-labs/firefly-cli/internal/docker"
	"github.com/hyperledger-labs/firefly-cli/pkg/types"
)

func GenerateCryptoMaterial(cryptogenConfigPath string, outputPath string, verbose bool) error {
	// Use cryptogen in the hyperledger/fabric-tools image to create the crypto material
	return docker.RunDockerCommand(path.Dir(cryptogenConfigPath), verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/etc/template.yml", cryptogenConfigPath), "-v", fmt.Sprintf("%s:/output", outputPath), "hyperledger/fabric-tools", "cryptogen", "generate", "--config", "/etc/template.yml", "--output", "/output")
}

func GenerateGenesisBlock(outputPath string, verbose bool) error {
	// Use configtxgen in the hyperledger/fabric-tools image to generate the genesis config
	return docker.RunDockerCommand(outputPath, verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/genesis", outputPath), "hyperledger/fabric-tools", "configtxgen", "-outputBlock", "/genesis/genesis_block.pb", "-profile", "SampleDevModeSolo", "-channelID", "firefly")
}

func GenerateDockerServiceDefinitions(s *types.Stack) []*docker.ServiceDefinition {

	stackDir := path.Join(constants.StacksDir, s.Name)
	serviceDefinitions := make([]*docker.ServiceDefinition, len(s.Members)+3)

	// Fabric CA
	serviceDefinitions[0] = &docker.ServiceDefinition{
		ServiceName: "ca_org1",
		Service: &docker.Service{
			Image: "hyperledger/fabric-ca:latest",
			Environment: map[string]string{
				"FABRIC_CA_HOME":                            "/etc/hyperledger/fabric-ca-server",
				"FABRIC_CA_SERVER_CA_NAME":                  "ca-org1",
				"FABRIC_CA_SERVER_TLS_ENABLED":              "true",
				"FABRIC_CA_SERVER_PORT":                     "7054",
				"FABRIC_CA_SERVER_OPERATIONS_LISTENADDRESS": "0.0.0.0:17054",
			},
			// TODO: Figure out how to increment ports here
			Ports: []string{
				"7054:7054",
				"17054:17054",
			},
			Command: "sh -c 'fabric-ca-server start -b admin:adminpw -d'",
		},
	}

	// Fabric Orderer
	serviceDefinitions[1] = &docker.ServiceDefinition{
		ServiceName: "orderer.example.com",
		Service: &docker.Service{
			Image: "hyperledger/fabric-ca:latest",
			Environment: map[string]string{
				"FABRIC_LOGGING_SPEC":                       "INFO",
				"ORDERER_GENERAL_LISTENADDRESS":             "0.0.0.0",
				"ORDERER_GENERAL_LISTENPORT":                "7050",
				"ORDERER_GENERAL_LOCALMSPID":                "OrdererMSP",
				"ORDERER_GENERAL_LOCALMSPDIR":               "/var/hyperledger/orderer/msp",
				"ORDERER_GENERAL_TLS_ENABLED":               "true",
				"ORDERER_GENERAL_TLS_PRIVATEKEY":            "/var/hyperledger/orderer/tls/server.key",
				"ORDERER_GENERAL_TLS_CERTIFICATE":           "/var/hyperledger/orderer/tls/server.crt",
				"ORDERER_GENERAL_TLS_ROOTCAS":               "[/var/hyperledger/orderer/tls/ca.crt]",
				"ORDERER_KAFKA_TOPIC_REPLICATIONFACTOR":     "1",
				"ORDERER_KAFKA_VERBOSE":                     "true",
				"ORDERER_GENERAL_CLUSTER_CLIENTCERTIFICATE": "/var/hyperledger/orderer/tls/server.crt",
				"ORDERER_GENERAL_CLUSTER_CLIENTPRIVATEKEY":  "/var/hyperledger/orderer/tls/server.key",
				"ORDERER_GENERAL_CLUSTER_ROOTCAS":           "[/var/hyperledger/orderer/tls/ca.crt]",
				"ORDERER_GENERAL_BOOTSTRAPMETHOD":           "none",
				"ORDERER_CHANNELPARTICIPATION_ENABLED":      "true",
				"ORDERER_ADMIN_TLS_ENABLED":                 "true",
				"ORDERER_ADMIN_TLS_CERTIFICATE":             "/var/hyperledger/orderer/tls/server.crt",
				"ORDERER_ADMIN_TLS_PRIVATEKEY":              "/var/hyperledger/orderer/tls/server.key",
				"ORDERER_ADMIN_TLS_ROOTCAS":                 "[/var/hyperledger/orderer/tls/ca.crt]",
				"ORDERER_ADMIN_TLS_CLIENTROOTCAS":           "[/var/hyperledger/orderer/tls/ca.crt]",
				"ORDERER_ADMIN_LISTENADDRESS":               "0.0.0.0:7053",
				"ORDERER_OPERATIONS_LISTENADDRESS":          "0.0.0.0:17050",
			},
			WorkingDir: "/opt/gopath/src/github.com/hyperledger/fabric",
			Command:    "orderer",
			Volumes: []string{
				fmt.Sprintf("%s:/var/hyperledger/orderer/orderer.genesis.block", path.Join(stackDir, "blockchain", "genesis_block.pb")),
				fmt.Sprintf("%s:/var/hyperledger/orderer/msp", path.Join(stackDir, "blockchain", "cryptogen", "ordererOrganizations", "example.com", "orderers", "orderer.example.com", "msp")),
				fmt.Sprintf("%s:/var/hyperledger/orderer/tls", path.Join(stackDir, "blockchain", "cryptogen", "ordererOrganizations", "example.com", "orderers", "orderer.example.com", "tls")),
				"orderer.example.com:/var/hyperledger/production/orderer",
			},
			// TODO: Figure out how to increment ports here
			Ports: []string{
				"7054:7054",
				"17054:17054",
			},
		},
		VolumeNames: []string{"orderer.example.com"},
	}

	// Fabric Peer
	serviceDefinitions[2] = &docker.ServiceDefinition{
		ServiceName: "peer0.org1.example.com",
		Service: &docker.Service{
			Image: "hyperledger/fabric-peer:latest",
			Environment: map[string]string{
				"CORE_VM_ENDPOINT":                      "unix:///host/var/run/docker.sock",
				"CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE": "fabric_test",
				"FABRIC_LOGGING_SPEC":                   "INFO",
				"CORE_PEER_TLS_ENABLED":                 "true",
				"CORE_PEER_PROFILE_ENABLED":             "false",
				"CORE_PEER_TLS_CERT_FILE":               "/etc/hyperledger/fabric/tls/server.crt",
				"CORE_PEER_TLS_KEY_FILE":                "/etc/hyperledger/fabric/tls/server.key",
				"CORE_PEER_TLS_ROOTCERT_FILE":           "/etc/hyperledger/fabric/tls/ca.crt",
				"CORE_PEER_ID":                          "peer0.org1.example.com",
				"CORE_PEER_ADDRESS":                     "peer0.org1.example.com:7051",
				"CORE_PEER_LISTENADDRESS":               "0.0.0.0:7051",
				"CORE_PEER_CHAINCODEADDRESS":            "peer0.org1.example.com:7052",
				"CORE_PEER_CHAINCODELISTENADDRESS":      "0.0.0.0:7052",
				"CORE_PEER_GOSSIP_BOOTSTRAP":            "peer0.org1.example.com:7051",
				"CORE_PEER_GOSSIP_EXTERNALENDPOINT":     "peer0.org1.example.com:7051",
				"CORE_PEER_LOCALMSPID":                  "Org1MSP",
				"CORE_OPERATIONS_LISTENADDRESS":         "0.0.0.0:17051",
			},
			Volumes: []string{
				fmt.Sprintf("%s:/etc/hyperledger/fabric/msp", path.Join(stackDir, "blockchain", "cryptogen", "peerOrganizations", "org1.example.com", "peers", "peer0.org1.example.com", "msp")),
				fmt.Sprintf("%s:/etc/hyperledger/fabric/tls", path.Join(stackDir, "blockchain", "cryptogen", "peerOrganizations", "org1.example.com", "peers", "peer0.org1.example.com", "tls")),
				"peer0.org1.example.com:/var/hyperledger/production",
			},
		},
	}

	// Fabconnect instance for each member
	for i, member := range s.Members {
		serviceDefinitions[i+3] = &docker.ServiceDefinition{
			ServiceName: fmt.Sprintf("fabconnect_%s", member.ID),
		}
	}

	return serviceDefinitions
}

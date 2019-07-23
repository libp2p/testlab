package ipfs

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	napi "github.com/hashicorp/nomad/api"
	ci "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
)

var mergeConfigMemo = map[string][]byte{}

func mergeConfig(configPath string, values map[string]interface{}) ([]byte, error) {
	var (
		configBytes []byte
		ok          bool
		err         error
	)
	configBytes, ok = mergeConfigMemo[configPath]
	if !ok {
		configBytes, err = ioutil.ReadFile(configPath)
		if err != nil {
			return nil, err
		}
		mergeConfigMemo[configPath] = configBytes
	}

	config := map[string]interface{}{}
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return nil, err
	}

	for k, v := range values {
		config[k] = v
	}

	return json.MarshalIndent(config, "", "  ")
}

func generateIpfsKeys() (string, string, error) {
	sk, pk, err := ci.GenerateKeyPair(ci.RSA, 2048)
	if err != nil {
		return "", "", err
	}
	skbytes, err := sk.Bytes()
	if err != nil {
		return "", "", err
	}
	skstring := base64.StdEncoding.EncodeToString(skbytes)
	id, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return "", "", err
	}
	idstring := id.Pretty()

	return skstring, idstring, nil
}

func ipfsConfiguration(root, containerRoot string, bootstrappers []string) ([]*napi.Template, error) {
	infos, err := ioutil.ReadDir(root)
	if err != nil {
		return nil, err
	}

	// Generate config vars
	skstring, idstring, err := generateIpfsKeys()
	if err != nil {
		return nil, err
	}
	ipfsConfig := map[string]interface{}{
		"Bootstrap": bootstrappers,
		"Identity": map[string]interface{}{
			"PrivKey": skstring,
			"PeerID":  idstring,
		},
		"Addresses": map[string]interface{}{
			"Swarm": []interface{}{
				`/ip4/{{env %%NOMAD_IP_libp2p%%}}/tcp/{{env %%NOMAD_PORT_libp2p%%}}`,
			},
			"Announce":   []interface{}{},
			"NoAnnounce": []interface{}{},
			"API":        `/ip4/{{env %%NOMAD_IP_ipfsapi%%}}/tcp/{{env %%NOMAD_PORT_ipfsapi%%}}`,
			"Gateway":    `/ip4/{{env %%NOMAD_IP_ipfsgateway%%}}/tcp/{{env %%NOMAD_PORT_ipfsgateway%%}}`,
		},
	}

	// Render templates
	var templates []*napi.Template
	for _, info := range infos {
		if info.Name() == "config" {
			configPath := filepath.Join(root, "config")
			ipfsConfigBytes, err := mergeConfig(configPath, ipfsConfig)
			if err != nil {
				return nil, err
			}
			ipfsConfigString := string(ipfsConfigBytes)
			// TODO: don't do this
			ipfsConfigString = strings.Replace(ipfsConfigString, "%%", `"`, -1)
			fmt.Println(ipfsConfigString)
			destPath := filepath.Join(containerRoot, info.Name())
			template := &napi.Template{
				EmbeddedTmpl: &ipfsConfigString,
				DestPath:     &destPath,
			}
			templates = append(templates, template)
		} else {
			templateBytes, err := ioutil.ReadFile(filepath.Join(root, info.Name()))
			if err != nil {
				return nil, err
			}
			templateString := string(templateBytes)
			destPath := filepath.Join(containerRoot, info.Name())
			template := &napi.Template{
				EmbeddedTmpl: &templateString,
				DestPath:     &destPath,
			}
			templates = append(templates, template)
		}
	}

	return templates, nil
}

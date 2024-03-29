// Copyright 2023 Antrea Authors
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

package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	anomalydetector "antrea.io/theia/pkg/apis/intelligence/v1alpha1"
	"antrea.io/theia/pkg/theia/portforwarder"
)

func TestAnomalyDetectionList(t *testing.T) {
	testCases := []struct {
		name             string
		testServer       *httptest.Server
		expectedMsg      []string
		expectedErrorMsg string
	}{
		{
			name: "Valid case",
			testServer: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch strings.TrimSpace(r.URL.Path) {
				case "/apis/intelligence.theia.antrea.io/v1alpha1/throughputanomalydetectors":
					tadList := &anomalydetector.ThroughputAnomalyDetectorList{
						Items: []anomalydetector.ThroughputAnomalyDetector{
							{
								ObjectMeta: metav1.ObjectMeta{
									Name: "tad-test1",
								},
								Status: anomalydetector.ThroughputAnomalyDetectorStatus{
									SparkApplication: "test1",
								}},
						},
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(tadList)
				}
			})),
			expectedMsg:      []string{"tad-test1"},
			expectedErrorMsg: "",
		},
		{
			name: "ThroughputAnomalyDetectionList not found",
			testServer: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch strings.TrimSpace(r.URL.Path) {
				case "/apis/intelligence.theia.antrea.io/v1alpha1/throughputanomalydetectors":
					http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				}
			})),
			expectedMsg:      []string{},
			expectedErrorMsg: "error when getting anomaly detection job list:",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			defer tt.testServer.Close()
			oldFunc := SetupTheiaClientAndConnection
			SetupTheiaClientAndConnection = func(cmd *cobra.Command, useClusterIP bool) (restclient.Interface, *portforwarder.PortForwarder, error) {
				clientConfig := &restclient.Config{Host: tt.testServer.URL, TLSClientConfig: restclient.TLSClientConfig{Insecure: true}}
				clientset, _ := kubernetes.NewForConfig(clientConfig)
				return clientset.CoreV1().RESTClient(), nil, nil
			}
			defer func() {
				SetupTheiaClientAndConnection = oldFunc
			}()
			cmd := new(cobra.Command)
			cmd.Flags().Bool("use-cluster-ip", true, "")

			orig := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			err := anomalyDetectionList(cmd, []string{})
			if tt.expectedErrorMsg == "" {
				assert.NoError(t, err)
				outcome := readStdout(t, r, w)
				os.Stdout = orig
				assert.Contains(t, outcome, "test1")
				for _, msg := range tt.expectedMsg {
					assert.Contains(t, outcome, msg)
				}
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
			}
		})
	}
}

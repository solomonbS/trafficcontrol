package v4

/*

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

import (
	"net/http"
	"testing"
	"time"

	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-util"
	client "github.com/apache/trafficcontrol/traffic_ops/v4-client"
)

var SteeringUserSession *client.Session

func TestSteeringTargets(t *testing.T) {

	WithObjs(t, []TCObj{CDNs, Types, Tenants, Parameters, Profiles, Statuses, Divisions, Regions, PhysLocations, CacheGroups, Servers, Topologies, ServiceCategories, DeliveryServices, Users, SteeringTargets}, func() {
		GetTestSteeringTargetsIMS(t)
		GetTestSteeringTargets(t)
		currentTime := time.Now().UTC().Add(-5 * time.Second)
		time := currentTime.Format(time.RFC1123)
		var header http.Header
		header = make(map[string][]string)
		header.Set(rfc.IfModifiedSince, time)
		header.Set(rfc.IfUnmodifiedSince, time)
		UpdateTestSteeringTargets(t)
		UpdateTestSteeringTargetsWithHeaders(t, header)
		GetTestSteeringTargetsIMSAfterChange(t, header)
		header = make(map[string][]string)
		etag := rfc.ETag(currentTime)
		header.Set(rfc.IfMatch, etag)
		UpdateTestSteeringTargetsWithHeaders(t, header)
	})

}

func UpdateTestSteeringTargetsWithHeaders(t *testing.T, header http.Header) {
	if len(testData.SteeringTargets) < 1 {
		t.Fatal("updating steering target: no steering target test data")
	}
	st := testData.SteeringTargets[0]
	if st.DeliveryService == nil {
		t.Fatal("updating steering target: test data missing ds")
	}
	if st.Target == nil {
		t.Fatal("updating steering target: test data missing target")
	}

	opts := client.NewRequestOptions()
	opts.Header = header
	opts.QueryParameters.Set("xmlId", string(*st.DeliveryService))
	respDS, _, err := SteeringUserSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("updating steering target: getting ds: %v - alerts: %+v", err, respDS.Alerts)
	}
	if len(respDS.Response) < 1 {
		t.Fatal("updating steering target: getting ds: not found")
	}
	if respDS.Response[0].ID == nil {
		t.Fatal("updating steering target: getting ds: nil id returned")
	}
	dsID := *respDS.Response[0].ID

	sts, _, err := SteeringUserSession.GetSteeringTargets(dsID, client.RequestOptions{})
	if err != nil {
		t.Fatalf("updating steering targets: getting steering target: %v - alerts: %+v", err, sts.Alerts)
	}
	if len(sts.Response) < 1 {
		t.Fatal("updating steering targets: getting steering target: got 0")
	}
	st = sts.Response[0]

	expected := util.JSONIntStr(-12345)
	if st.Value != nil && *st.Value == expected {
		expected++
	}
	st.Value = &expected

	opts.QueryParameters.Del("xmlId")
	_, reqInf, err := SteeringUserSession.UpdateSteeringTarget(st, opts)
	if err == nil {
		t.Errorf("Expected error about precondition failed, but got none")
	}
	if reqInf.StatusCode != http.StatusPreconditionFailed {
		t.Errorf("Expected status code 412, got %v", reqInf.StatusCode)
	}
}

func GetTestSteeringTargetsIMSAfterChange(t *testing.T, header http.Header) {
	if len(testData.SteeringTargets) < 1 {
		t.Fatal("updating steering target: no steering target test data")
	}
	st := testData.SteeringTargets[0]
	if st.DeliveryService == nil {
		t.Fatal("updating steering target: test data missing ds")
	}
	opts := client.NewRequestOptions()
	opts.Header = header
	opts.QueryParameters.Set("xmlId", string(*st.DeliveryService))
	respDS, reqInf, err := SteeringUserSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v - alerts: %+v", err, respDS.Alerts)
	}
	if reqInf.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 status code, got %v", reqInf.StatusCode)
	}

	currentTime := time.Now().UTC()
	currentTime = currentTime.Add(1 * time.Second)
	timeStr := currentTime.Format(time.RFC1123)

	opts.Header.Set(rfc.IfModifiedSince, timeStr)
	respDS, reqInf, err = SteeringUserSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v - alerts: %+v", err, respDS.Alerts)
	}
	if reqInf.StatusCode != http.StatusNotModified {
		t.Fatalf("Expected 304 status code, got %v", reqInf.StatusCode)
	}
}

func GetTestSteeringTargetsIMS(t *testing.T) {
	if len(testData.SteeringTargets) < 1 {
		t.Fatal("updating steering target: no steering target test data")
	}
	st := testData.SteeringTargets[0]
	if st.DeliveryService == nil {
		t.Fatal("updating steering target: test data missing ds")
	}

	futureTime := time.Now().AddDate(0, 0, 1)
	time := futureTime.Format(time.RFC1123)

	opts := client.NewRequestOptions()
	opts.Header.Set(rfc.IfModifiedSince, time)
	opts.QueryParameters.Set("xmlId", string(*st.DeliveryService))
	respDS, reqInf, err := SteeringUserSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v - alerts: %+v", err, respDS.Alerts)
	}
	if reqInf.StatusCode != http.StatusNotModified {
		t.Fatalf("Expected 304 status code, got %v", reqInf.StatusCode)
	}
}

// SetupSteeringTargets calls the CreateSteeringTargets test. It also sets the steering user session
// with the logged in steering user. SteeringUserSession is used by steering target test functions.
// Running this function depends on CreateTestUsers.
func SetupSteeringTargets(t *testing.T) {
	var err error
	toReqTimeout := time.Second * time.Duration(Config.Default.Session.TimeoutInSecs)
	SteeringUserSession, _, err = client.LoginWithAgent(TOSession.URL, "steering", "pa$$word", true, "to-api-v1-client-tests/steering", true, toReqTimeout)
	if err != nil {
		t.Fatalf("failed to get log in with steering user: %v", err.Error())
	}

	CreateTestSteeringTargets(t)
}

func CreateTestSteeringTargets(t *testing.T) {
	for _, st := range testData.SteeringTargets {
		if st.Type == nil {
			t.Fatal("creating steering target: test data missing type")
		}
		if st.DeliveryService == nil {
			t.Fatal("creating steering target: test data missing ds")
		}
		if st.Target == nil {
			t.Fatal("creating steering target: test data missing target")
		}

		{
			opts := client.NewRequestOptions()
			opts.QueryParameters.Set("name", *st.Type)
			respTypes, _, err := SteeringUserSession.GetTypes(opts)
			if err != nil {
				t.Fatalf("creating steering target: getting Type: %v - alerts: %+v", err, respTypes.Alerts)
			} else if len(respTypes.Response) < 1 {
				t.Fatal("creating steering target: getting Type: not found")
			}
			st.TypeID = util.IntPtr(respTypes.Response[0].ID)
		}
		{
			opts := client.NewRequestOptions()
			opts.QueryParameters.Set("xmlId", string(*st.DeliveryService))
			respDS, _, err := SteeringUserSession.GetDeliveryServices(opts)
			if err != nil {
				t.Fatalf("creating steering target: getting ds: %v - alerts: %+v", err, respDS.Alerts)
			} else if len(respDS.Response) < 1 {
				t.Fatal("creating steering target: getting ds: not found")
			} else if respDS.Response[0].ID == nil {
				t.Fatal("creating steering target: getting ds: nil ID returned")
			}
			dsID := uint64(*respDS.Response[0].ID)
			st.DeliveryServiceID = &dsID
		}
		{
			opts := client.NewRequestOptions()
			opts.QueryParameters.Set("xmlId", string(*st.Target))
			respTarget, _, err := SteeringUserSession.GetDeliveryServices(opts)
			if err != nil {
				t.Fatalf("creating steering target: getting target ds: %v - alerts: %+v", err, respTarget.Alerts)
			} else if len(respTarget.Response) < 1 {
				t.Fatal("creating steering target: getting target ds: not found")
			} else if respTarget.Response[0].ID == nil {
				t.Fatal("creating steering target: getting target ds: nil ID returned")
			}
			targetID := uint64(*respTarget.Response[0].ID)
			st.TargetID = &targetID
		}

		resp, _, err := SteeringUserSession.CreateSteeringTarget(st, client.RequestOptions{})
		if err != nil {
			t.Fatalf("creating steering target: %v - alerts: %+v", err, resp.Alerts)
		}
	}
}

func UpdateTestSteeringTargets(t *testing.T) {
	if len(testData.SteeringTargets) < 1 {
		t.Fatal("updating steering target: no steering target test data")
	}
	st := testData.SteeringTargets[0]
	if st.DeliveryService == nil {
		t.Fatal("updating steering target: test data missing ds")
	}
	if st.Target == nil {
		t.Fatal("updating steering target: test data missing target")
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("xmlId", string(*st.DeliveryService))
	respDS, _, err := SteeringUserSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("updating steering target: getting ds: %v - alerts: %+v", err, respDS.Alerts)
	}
	if len(respDS.Response) < 1 {
		t.Fatal("updating steering target: getting ds: not found")
	}
	if respDS.Response[0].ID == nil {
		t.Fatal("updating steering target: getting ds: nil id returned")
	}
	dsID := *respDS.Response[0].ID

	sts, _, err := SteeringUserSession.GetSteeringTargets(dsID, client.RequestOptions{})
	if err != nil {
		t.Fatalf("updating steering targets: getting steering target: %v - alerts: %+v", err, sts.Alerts)
	}
	if len(sts.Response) < 1 {
		t.Fatal("updating steering targets: getting steering target: got 0")
	}
	st = sts.Response[0]

	expected := util.JSONIntStr(-12345)
	if st.Value != nil && *st.Value == expected {
		expected++
	}
	st.Value = &expected

	alerts, _, err := SteeringUserSession.UpdateSteeringTarget(st, client.RequestOptions{})
	if err != nil {
		t.Fatalf("updating steering targets: updating: %v - alerts: %+v", err, alerts.Alerts)
	}

	sts, _, err = SteeringUserSession.GetSteeringTargets(dsID, client.RequestOptions{})
	if err != nil {
		t.Fatalf("updating steering targets: getting updated steering target: %v - alerts: %+v", err, sts.Alerts)
	}
	if len(sts.Response) < 1 {
		t.Fatal("updating steering targets: getting updated steering target: got 0")
	}
	actual := sts.Response[0]

	if actual.DeliveryServiceID == nil {
		t.Fatalf("steering target update: ds id expected %v actual %v", dsID, nil)
	} else if *actual.DeliveryServiceID != uint64(dsID) {
		t.Fatalf("steering target update: ds id expected %v actual %v", dsID, *actual.DeliveryServiceID)
	}
	if actual.TargetID == nil {
		t.Fatalf("steering target update: ds id expected %v actual %v", dsID, nil)
	} else if *actual.TargetID != *st.TargetID {
		t.Fatalf("steering target update: ds id expected %v actual %v", *st.TargetID, *actual.TargetID)
	}
	if actual.TypeID == nil {
		t.Fatalf("steering target update: ds id expected %v actual %v", *st.TypeID, nil)
	} else if *actual.TypeID != *st.TypeID {
		t.Fatalf("steering target update: ds id expected %v actual %v", *st.TypeID, *actual.TypeID)
	}
	if actual.DeliveryService == nil {
		t.Fatalf("steering target update: ds expected %v actual %v", *st.DeliveryService, nil)
	} else if *st.DeliveryService != *actual.DeliveryService {
		t.Fatalf("steering target update: ds name expected %v actual %v", *st.DeliveryService, *actual.DeliveryService)
	}
	if actual.Target == nil {
		t.Fatalf("steering target update: target expected %v actual %v", *st.Target, nil)
	} else if *st.Target != *actual.Target {
		t.Fatalf("steering target update: target expected %v actual %v", *st.Target, *actual.Target)
	}
	if actual.Type == nil {
		t.Fatalf("steering target update: type expected %v actual %v", *st.Type, nil)
	} else if *st.Type != *actual.Type {
		t.Fatalf("steering target update: type expected %v actual %v", *st.Type, *actual.Type)
	}
	if actual.Value == nil {
		t.Fatalf("steering target update: ds expected %v actual %v", *st.Value, nil)
	} else if *st.Value != *actual.Value {
		t.Fatalf("steering target update: value expected %v actual %v", *st.Value, actual.Value)
	}
}

func GetTestSteeringTargets(t *testing.T) {
	if len(testData.SteeringTargets) < 1 {
		t.Fatal("updating steering target: no steering target test data")
	}
	st := testData.SteeringTargets[0]
	if st.DeliveryService == nil {
		t.Fatal("updating steering target: test data missing ds")
	}

	opts := client.NewRequestOptions()
	opts.QueryParameters.Set("xmlId", string(*st.DeliveryService))
	respDS, _, err := SteeringUserSession.GetDeliveryServices(opts)
	if err != nil {
		t.Fatalf("creating steering target: getting ds: %v - alerts: %+v", err, respDS.Alerts)
	} else if len(respDS.Response) < 1 {
		t.Fatal("steering target get: getting ds: not found")
	} else if respDS.Response[0].ID == nil {
		t.Fatal("steering target get: getting ds: nil id returned")
	}
	dsID := *respDS.Response[0].ID

	sts, _, err := SteeringUserSession.GetSteeringTargets(dsID, client.RequestOptions{})
	if err != nil {
		t.Fatalf("steering target get: getting steering target: %v - alerts: %+v", err, sts.Alerts)
	}

	if len(sts.Response) != len(testData.SteeringTargets) {
		t.Fatalf("steering target get: expected %d actual %d", len(testData.SteeringTargets), len(sts.Response))
	}

	expected := testData.SteeringTargets[0]
	actual := sts.Response[0]

	if actual.DeliveryServiceID == nil {
		t.Fatalf("steering target get: ds id expected %v actual %v", dsID, nil)
	} else if *actual.DeliveryServiceID != uint64(dsID) {
		t.Fatalf("steering target get: ds id expected %v actual %v", dsID, *actual.DeliveryServiceID)
	}
	if actual.DeliveryService == nil {
		t.Fatalf("steering target get: ds expected %v actual %v", expected.DeliveryService, nil)
	} else if *expected.DeliveryService != *actual.DeliveryService {
		t.Fatalf("steering target get: ds name expected %v actual %v", expected.DeliveryService, actual.DeliveryService)
	}
	if actual.Target == nil {
		t.Fatalf("steering target get: target expected %v actual %v", expected.Target, nil)
	} else if *expected.Target != *actual.Target {
		t.Fatalf("steering target get: target expected %v actual %v", expected.Target, actual.Target)
	}
	if actual.Type == nil {
		t.Fatalf("steering target get: type expected %v actual %v", expected.Type, nil)
	} else if *expected.Type != *actual.Type {
		t.Fatalf("steering target get: type expected %v actual %v", expected.Type, actual.Type)
	}
	if actual.Value == nil {
		t.Fatalf("steering target get: ds expected %v actual %v", expected.Value, nil)
	} else if *expected.Value != *actual.Value {
		t.Fatalf("steering target get: value expected %v actual %v", *expected.Value, *actual.Value)
	}
}

func DeleteTestSteeringTargets(t *testing.T) {
	dsIDs := []uint64{}
	for _, st := range testData.SteeringTargets {
		if st.DeliveryService == nil {
			t.Fatal("deleting steering target: test data missing ds")
		}
		if st.Target == nil {
			t.Fatal("deleting steering target: test data missing target")
		}

		opts := client.NewRequestOptions()
		opts.QueryParameters.Set("xmlId", string(*st.DeliveryService))
		respDS, _, err := SteeringUserSession.GetDeliveryServices(opts)
		if err != nil {
			t.Fatalf("deleting steering target: getting ds: %v - alerts: %+v", err, respDS.Alerts)
		} else if len(respDS.Response) < 1 {
			t.Fatal("deleting steering target: getting ds: not found")
		} else if respDS.Response[0].ID == nil {
			t.Fatal("deleting steering target: getting ds: nil ID returned")
		}
		dsID := uint64(*respDS.Response[0].ID)
		st.DeliveryServiceID = &dsID

		dsIDs = append(dsIDs, dsID)

		opts.QueryParameters.Set("xmlId", string(*st.Target))
		respTarget, _, err := SteeringUserSession.GetDeliveryServices(opts)
		if err != nil {
			t.Fatalf("deleting steering target: getting target ds: %v - alerts: %+v", err, respTarget.Alerts)
		} else if len(respTarget.Response) < 1 {
			t.Fatal("deleting steering target: getting target ds: not found")
		} else if respTarget.Response[0].ID == nil {
			t.Fatal("deleting steering target: getting target ds: not found")
		}
		targetID := uint64(*respTarget.Response[0].ID)
		st.TargetID = &targetID

		resp, _, err := SteeringUserSession.DeleteSteeringTarget(int(*st.DeliveryServiceID), int(*st.TargetID), client.RequestOptions{})
		if err != nil {
			t.Fatalf("deleting steering target: deleting: %v - alerts: %+v", err, resp.Alerts)
		}
	}

	for _, dsID := range dsIDs {
		sts, _, err := SteeringUserSession.GetSteeringTargets(int(dsID), client.RequestOptions{})
		if err != nil {
			t.Fatalf("deleting steering targets: getting steering target: %v - alerts: %+v", err, sts.Alerts)
		}
		if len(sts.Response) != 0 {
			t.Fatalf("deleting steering targets: after delete, getting steering target: expected 0 actual %d", len(sts.Response))
		}
	}
}

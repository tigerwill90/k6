/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2021 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package http

import (
	"context"
	"testing"

	"github.com/dop251/goja"
	"github.com/loadimpact/k6/js/common"
	"github.com/loadimpact/k6/lib/metrics"
	"github.com/loadimpact/k6/stats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpectedStatuses(t *testing.T) {
	t.Parallel()
	rt := goja.New()
	rt.SetFieldNameMapper(common.FieldNameMapper{})
	ctx := context.Background()

	ctx = common.WithRuntime(ctx, rt)
	rt.Set("http", common.Bind(rt, New(), &ctx))
	cases := map[string]struct {
		code, err string
		expected  expectedStatuses
	}{
		"good example": {
			expected: expectedStatuses{exact: []int{200, 300}, minmax: [][2]int{{200, 300}}},
			code:     `(http.expectedStatuses(200, 300, {min: 200, max:300}))`,
		},

		"strange example": {
			expected: expectedStatuses{exact: []int{200, 300}, minmax: [][2]int{{200, 300}}},
			code:     `(http.expectedStatuses(200, 300, {min: 200, max:300, other: "attribute"}))`,
		},

		"string status code": {
			code: `(http.expectedStatuses(200, "300", {min: 200, max:300}))`,
			err:  "argument number 2 to expectedStatuses was neither an integer nor an object like {min:100, max:329}",
		},

		"string max status code": {
			code: `(http.expectedStatuses(200, 300, {min: 200, max:"300"}))`,
			err:  "both min and max need to be number for argument number 3",
		},
		"float status code": {
			err:  "argument number 2 to expectedStatuses was neither an integer nor an object like {min:100, max:329}",
			code: `(http.expectedStatuses(200, 300.5, {min: 200, max:300}))`,
		},

		"float max status code": {
			err:  "both min and max need to be number for argument number 3",
			code: `(http.expectedStatuses(200, 300, {min: 200, max:300.5}))`,
		},
		"no arguments": {
			code: `(http.expectedStatuses())`,
			err:  "no arguments",
		},
	}

	for name, testCase := range cases {
		name, testCase := name, testCase
		t.Run(name, func(t *testing.T) {
			val, err := rt.RunString(testCase.code)
			if testCase.err == "" {
				require.NoError(t, err)
				got := new(expectedStatuses)
				err = rt.ExportTo(val, &got)
				require.NoError(t, err)
				require.Equal(t, testCase.expected, *got)
				return // the t.Run
			}

			require.Error(t, err)
			exc := err.(*goja.Exception)
			require.Contains(t, exc.Error(), testCase.err)
		})
	}
}

type expectedSample struct {
	tags    map[string]string
	metrics []*stats.Metric
}

func TestResponseCallbackInAction(t *testing.T) {
	t.Parallel()
	tb, state, samples, rt, _ := newRuntime(t)
	defer tb.Cleanup()
	sr := tb.Replacer.Replace
	allHTTPMetrics := []*stats.Metric{
		metrics.HTTPReqs,
		metrics.HTTPReqFailed,
		metrics.HTTPReqBlocked,
		metrics.HTTPReqConnecting,
		metrics.HTTPReqDuration,
		metrics.HTTPReqReceiving,
		metrics.HTTPReqSending,
		metrics.HTTPReqWaiting,
		metrics.HTTPReqTLSHandshaking,
	}

	HTTPMetricsWithoutFailed := []*stats.Metric{
		metrics.HTTPReqs,
		metrics.HTTPReqBlocked,
		metrics.HTTPReqConnecting,
		metrics.HTTPReqDuration,
		metrics.HTTPReqReceiving,
		metrics.HTTPReqWaiting,
		metrics.HTTPReqSending,
		metrics.HTTPReqTLSHandshaking,
	}
	testCases := map[string]struct {
		code            string
		expectedSamples []expectedSample
	}{
		"basic": {
			code: `http.request("GET", "HTTPBIN_URL/redirect/1");`,
			expectedSamples: []expectedSample{
				{
					tags: map[string]string{
						"method": "GET",
						"url":    sr("HTTPBIN_URL/redirect/1"),
						"name":   sr("HTTPBIN_URL/redirect/1"),
						"status": "302",
						"group":  "",
						"passed": "true",
						"proto":  "HTTP/1.1",
					},
					metrics: allHTTPMetrics,
				},
				{
					tags: map[string]string{
						"method": "GET",
						"url":    sr("HTTPBIN_URL/get"),
						"name":   sr("HTTPBIN_URL/get"),
						"status": "200",
						"group":  "",
						"passed": "true",
						"proto":  "HTTP/1.1",
					},
					metrics: allHTTPMetrics,
				},
			},
		},
		"overwrite per request": {
			code: `
			http.setResponseCallback(http.expectedStatuses(200));
			res = http.request("GET", "HTTPBIN_URL/redirect/1");
			`,
			expectedSamples: []expectedSample{
				{
					tags: map[string]string{
						"method": "GET",
						"url":    sr("HTTPBIN_URL/redirect/1"),
						"name":   sr("HTTPBIN_URL/redirect/1"),
						"status": "302",
						"group":  "",
						"passed": "false", // this is on purpose
						"proto":  "HTTP/1.1",
					},
					metrics: allHTTPMetrics,
				},
				{
					tags: map[string]string{
						"method": "GET",
						"url":    sr("HTTPBIN_URL/get"),
						"name":   sr("HTTPBIN_URL/get"),
						"status": "200",
						"group":  "",
						"passed": "true",
						"proto":  "HTTP/1.1",
					},
					metrics: allHTTPMetrics,
				},
			},
		},

		"global overwrite": {
			code: `http.request("GET", "HTTPBIN_URL/redirect/1", null, {responseCallback: http.expectedStatuses(200)});`,
			expectedSamples: []expectedSample{
				{
					tags: map[string]string{
						"method": "GET",
						"url":    sr("HTTPBIN_URL/redirect/1"),
						"name":   sr("HTTPBIN_URL/redirect/1"),
						"status": "302",
						"group":  "",
						"passed": "false", // this is on purpose
						"proto":  "HTTP/1.1",
					},
					metrics: allHTTPMetrics,
				},
				{
					tags: map[string]string{
						"method": "GET",
						"url":    sr("HTTPBIN_URL/get"),
						"name":   sr("HTTPBIN_URL/get"),
						"status": "200",
						"group":  "",
						"passed": "true",
						"proto":  "HTTP/1.1",
					},
					metrics: allHTTPMetrics,
				},
			},
		},
		"per request overwrite with null": {
			code: `http.request("GET", "HTTPBIN_URL/redirect/1", null, {responseCallback: null});`,
			expectedSamples: []expectedSample{
				{
					tags: map[string]string{
						"method": "GET",
						"url":    sr("HTTPBIN_URL/redirect/1"),
						"name":   sr("HTTPBIN_URL/redirect/1"),
						"status": "302",
						"group":  "",
						"proto":  "HTTP/1.1",
					},
					metrics: HTTPMetricsWithoutFailed,
				},
				{
					tags: map[string]string{
						"method": "GET",
						"url":    sr("HTTPBIN_URL/get"),
						"name":   sr("HTTPBIN_URL/get"),
						"status": "200",
						"group":  "",
						"proto":  "HTTP/1.1",
					},
					metrics: HTTPMetricsWithoutFailed,
				},
			},
		},
		"global overwrite with null": {
			code: `
			http.setResponseCallback(null);
			res = http.request("GET", "HTTPBIN_URL/redirect/1");
			`,
			expectedSamples: []expectedSample{
				{
					tags: map[string]string{
						"method": "GET",
						"url":    sr("HTTPBIN_URL/redirect/1"),
						"name":   sr("HTTPBIN_URL/redirect/1"),
						"status": "302",
						"group":  "",
						"proto":  "HTTP/1.1",
					},
					metrics: HTTPMetricsWithoutFailed,
				},
				{
					tags: map[string]string{
						"method": "GET",
						"url":    sr("HTTPBIN_URL/get"),
						"name":   sr("HTTPBIN_URL/get"),
						"status": "200",
						"group":  "",
						"proto":  "HTTP/1.1",
					},
					metrics: HTTPMetricsWithoutFailed,
				},
			},
		},
	}
	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			state.HTTPResponseCallback = DefaultHTTPResponseCallback()

			_, err := rt.RunString(sr(testCase.code))
			assert.NoError(t, err)
			bufSamples := stats.GetBufferedSamples(samples)

			reqsCount := 0
			for _, container := range bufSamples {
				for _, sample := range container.GetSamples() {
					if sample.Metric.Name == "http_reqs" {
						reqsCount++
					}
				}
			}

			require.Equal(t, len(testCase.expectedSamples), reqsCount)

			for i, expectedSample := range testCase.expectedSamples {
				assertRequestMetricsEmittedSingle(t, bufSamples[i], expectedSample.tags, expectedSample.metrics)
			}
		})
	}
}

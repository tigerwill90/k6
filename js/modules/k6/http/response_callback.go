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
	"errors"
	"fmt"

	"github.com/dop251/goja"
	"github.com/loadimpact/k6/js/common"
	"github.com/loadimpact/k6/lib"
)

//nolint:gochecknoglobals
var defaultExpectedStatuses = expectedStatuses{
	minmax: [][2]int{{200, 399}},
}

// DefaultHTTPResponseCallback ...
func DefaultHTTPResponseCallback() func(int) bool {
	return defaultExpectedStatuses.match
}

type expectedStatuses struct {
	minmax [][2]int
	exact  []int // this can be done with the above and vice versa
}

func (e expectedStatuses) match(status int) bool {
	for _, v := range e.exact { // binary search
		if v == status {
			return true
		}
	}

	for _, v := range e.minmax { // binary search
		if v[0] <= status && status <= v[1] {
			return true
		}
	}
	return false
}

// ExpectedStatuses is ...
func (*HTTP) ExpectedStatuses(ctx context.Context, args ...goja.Value) *expectedStatuses { //nolint: golint
	rt := common.GetRuntime(ctx)

	if len(args) == 0 {
		common.Throw(rt, errors.New("no arguments"))
	}
	var result expectedStatuses

	for i, arg := range args {
		o := arg.ToObject(rt)
		if o == nil {
			//nolint:lll
			common.Throw(rt, fmt.Errorf("argument number %d to expectedStatuses was neither an integer nor a an object like {min:100, max:329}", i+1))
		}

		if o.ClassName() == "Number" {
			result.exact = append(result.exact, int(o.ToInteger()))
		} else {
			min := o.Get("min")
			max := o.Get("max")
			if min == nil || max == nil {
				//nolint:lll
				common.Throw(rt, fmt.Errorf("argument number %d to expectedStatuses was neither an integer nor a an object like {min:100, max:329}", i+1))
			}
			if !(checkNumber(min, rt) && checkNumber(max, rt)) {
				common.Throw(rt, fmt.Errorf("both min and max need to be number for argument number %d", i+1))
			}

			result.minmax = append(result.minmax, [2]int{int(min.ToInteger()), int(max.ToInteger())})
		}
	}
	return &result
}

func checkNumber(a goja.Value, rt *goja.Runtime) bool {
	o := a.ToObject(rt)
	return o != nil && o.ClassName() == "Number"
}

// SetResponseCallback ..
func (h HTTP) SetResponseCallback(ctx context.Context, es *expectedStatuses) {
	if es != nil {
		lib.GetState(ctx).HTTPResponseCallback = es.match
	} else {
		lib.GetState(ctx).HTTPResponseCallback = nil
	}
}

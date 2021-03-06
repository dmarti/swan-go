/* ****************************************************************************
 * Copyright 2020 51 Degrees Mobile Experts Limited (51degrees.com)
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not
 * use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
 * License for the specific language governing permissions and limitations
 * under the License.
 * ***************************************************************************/

package swan

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"owid"
	"strings"
	"swift"
)

// AddHandlers adds swift and owid end points configured from the JSON file
// provided.
func AddHandlers(
	settingsFile string,
	swanAccess Access,
	swiftAccess swift.Access,
	owidAccess owid.Access,
	htmlTemplate string,
	malformedHandler func(w http.ResponseWriter, r *http.Request)) error {

	// Create the new set of services.
	s := newServices(settingsFile, swanAccess, swiftAccess, owidAccess)

	// Add the SWIFT handlers.
	swift.AddHandlers(s.swift, malformedHandler)

	// Add the OWID handlers.
	owid.AddHandlers(s.owid)

	// Add the SWAN handlers.
	http.HandleFunc("/swan/api/v1/fetch", handlerFetch(s))
	http.HandleFunc("/swan/api/v1/update", handlerUpdate(s))
	http.HandleFunc("/swan/api/v1/decode-as-json", handlerDecodeAsJSON(s))
	http.HandleFunc("/swan/api/v1/create-offer-id", handlerCreateOfferID(s))
	h, err := handlerCapture(s, htmlTemplate)
	if err != nil {
		return err
	}
	http.HandleFunc("/swan/preferences/", h)
	return nil
}

func newResponseError(url string, resp *http.Response) error {
	in, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return fmt.Errorf("API call '%s' returned '%d' and '%s'",
		url, resp.StatusCode, in)
}

func returnAPIError(
	c *Configuration,
	w http.ResponseWriter,
	err error,
	code int) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	http.Error(w, err.Error(), code)
	if c.Debug {
		println(err.Error())
	}
}

func returnRequestError(
	c *Configuration,
	w http.ResponseWriter,
	err error,
	code int) {
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if c.Debug {
		http.Error(w, err.Error(), code)
	} else {
		http.Error(w, "", code)
	}
	if c.Debug {
		println(err.Error())
	}
}

func returnServerError(c *Configuration, w http.ResponseWriter, err error) {
	w.Header().Set("Cache-Control", "no-cache")
	if c.Debug {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		http.Error(w, "", http.StatusInternalServerError)
	}
	if c.Debug {
		println(err.Error())
	}
}

// Removes white space from the HTML string provided whilst retaining valid
// HTML.
func removeHTMLWhiteSpace(h string) string {
	var sb strings.Builder
	for i, r := range h {

		// Only write out runes that are not control characters.
		if r != '\r' && r != '\n' && r != '\t' {

			// Only write this rune if the rune is not a space, or if it is a
			// space the preceding rune is not a space.
			if i == 0 || r != ' ' || h[i-1] != ' ' {
				sb.WriteRune(r)
			}
		}
	}
	return sb.String()
}

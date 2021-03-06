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
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"swift"
	"time"
)

func handlerDecodeAsJSON(s *services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var results *swift.Results

		// Check caller can access
		if s.getAccessAllowed(w, r) == false {
			returnAPIError(&s.config, w,
				errors.New("Not authorized"),
				http.StatusUnauthorized)
			return
		}

		// Get the form values from the input request.
		err := r.ParseForm()
		if err != nil {
			returnAPIError(&s.config, w, err, http.StatusInternalServerError)
			return
		}

		// Decrypt the string with the access node.
		in, err := decrypt(s, r.Form.Get("data"))
		if err != nil {
			returnAPIError(&s.config, w, err, http.StatusUnprocessableEntity)
			return
		}

		// Get the results.
		results, err = swift.DecodeResults(in)
		if err != nil {
			returnAPIError(&s.config, w, err, http.StatusUnprocessableEntity)
			return
		}

		// Change the values to OWIDs.
		for _, p := range results.Values {
			if p.Key == "email" {
				p.Key = "sid"
				p.Value, err = encodeAsOWID(s, r, createSID(p.Value))
			} else {
				p.Value, err = encodeAsOWID(s, r, []byte(p.Value))
			}
			if err != nil {
				returnAPIError(&s.config, w, err, http.StatusInternalServerError)
				return
			}
		}

		// Modify the expiry time.
		for _, i := range results.Values {
			i.Expires = time.Now().UTC().Add(time.Second * s.config.Timeout)
		}

		// Return the results as a JSON string.
		if err := json.NewEncoder(w).Encode(results.Values); err != nil {
			returnAPIError(&s.config, w, err, http.StatusUnprocessableEntity)
		}
	}
}

func decrypt(s *services, d string) ([]byte, error) {

	// Combine it with the access node to decrypt the result.
	u, err := url.Parse(s.config.Scheme + "://" + s.accessNode)
	if err != nil {
		return nil, err
	}
	u.Path = "/swift/api/v1/decrypt"
	q := u.Query()
	q.Set("data", d)
	q.Set("accessKey", s.config.AccessKey)
	u.RawQuery = q.Encode()

	// Call the URL and unpack the results if they're available.
	res, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, newResponseError(u.String(), res)
	}
	return ioutil.ReadAll(res.Body)
}

func encodeAsOWID(s *services, r *http.Request, v []byte) (string, error) {

	// Get the creator associated with this SWAN domain.
	c, err := s.owid.GetCreator(r.Host)
	if err != nil {
		return "", err
	}
	if c == nil {
		return "", fmt.Errorf(
			"No creator for '%s'. Use http[s]://%s/owid/register to setup "+
				"domain.",
			r.Host,
			r.Host)
	}

	// Create and sign the OWID.
	o := c.CreateOWID(v)
	err = c.Sign(o)
	if err != nil {
		return "", err
	}

	return o.AsBase64()
}

// TODO : What hashing algorithm do we want to use to turn email address into
// hashes?
func createSID(s string) []byte {
	hasher := sha1.New()
	hasher.Write([]byte(s))
	return hasher.Sum(nil)
}

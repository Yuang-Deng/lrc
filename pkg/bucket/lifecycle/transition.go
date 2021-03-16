/*
 * MinIO Cloud Storage, (C) 2019 MinIO, Inc.
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

package lifecycle

import (
	"encoding/xml"
	"time"
)

var (
	errTransitionInvalidDays     = Errorf("Days must be 0 or greater when used with Transition")
	errTransitionInvalidDate     = Errorf("Date must be provided in ISO 8601 format")
	errTransitionInvalid         = Errorf("Exactly one of Days (0 or greater) or Date (positive ISO 8601 format) should be present inside Expiration.")
	errTransitionDateNotMidnight = Errorf("'Date' must be at midnight GMT")
)

// TransitionDate is a embedded type containing time.Time to unmarshal
// Date in Transition
type TransitionDate struct {
	time.Time
}

// UnmarshalXML parses date from Transition and validates date format
func (tDate *TransitionDate) UnmarshalXML(d *xml.Decoder, startElement xml.StartElement) error {
	var dateStr string
	err := d.DecodeElement(&dateStr, &startElement)
	if err != nil {
		return err
	}
	// While AWS documentation mentions that the date specified
	// must be present in ISO 8601 format, in reality they allow
	// users to provide RFC 3339 compliant dates.
	trnDate, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return errTransitionInvalidDate
	}
	// Allow only date timestamp specifying midnight GMT
	hr, min, sec := trnDate.Clock()
	nsec := trnDate.Nanosecond()
	loc := trnDate.Location()
	if !(hr == 0 && min == 0 && sec == 0 && nsec == 0 && loc.String() == time.UTC.String()) {
		return errTransitionDateNotMidnight
	}

	*tDate = TransitionDate{trnDate}
	return nil
}

// MarshalXML encodes expiration date if it is non-zero and encodes
// empty string otherwise
func (tDate TransitionDate) MarshalXML(e *xml.Encoder, startElement xml.StartElement) error {
	if tDate.Time.IsZero() {
		return nil
	}
	return e.EncodeElement(tDate.Format(time.RFC3339), startElement)
}

// TransitionDays is a type alias to unmarshal Days in Transition
type TransitionDays int

// UnmarshalXML parses number of days from Transition and validates if
// >= 0
func (tDays *TransitionDays) UnmarshalXML(d *xml.Decoder, startElement xml.StartElement) error {
	var numDays int
	err := d.DecodeElement(&numDays, &startElement)
	if err != nil {
		return err
	}
	if numDays < 0 {
		return errTransitionInvalidDays
	}
	*tDays = TransitionDays(numDays)
	return nil
}

// MarshalXML encodes number of days to expire if it is non-zero and
// encodes empty string otherwise
func (tDays TransitionDays) MarshalXML(e *xml.Encoder, startElement xml.StartElement) error {
	if tDays == 0 {
		return nil
	}
	return e.EncodeElement(int(tDays), startElement)
}

// Transition - transition actions for a rule in lifecycle configuration.
type Transition struct {
	XMLName      xml.Name       `xml:"Transition"`
	Days         TransitionDays `xml:"Days,omitempty"`
	Date         TransitionDate `xml:"Date,omitempty"`
	StorageClass string         `xml:"StorageClass,omitempty"`

	set bool
}

// MarshalXML encodes transition field into an XML form.
func (t Transition) MarshalXML(enc *xml.Encoder, start xml.StartElement) error {
	if !t.set {
		return nil
	}
	type transitionWrapper Transition
	return enc.EncodeElement(transitionWrapper(t), start)
}

// UnmarshalXML decodes transition field from the XML form.
func (t *Transition) UnmarshalXML(d *xml.Decoder, startElement xml.StartElement) error {
	type transitionWrapper Transition
	var trw transitionWrapper
	err := d.DecodeElement(&trw, &startElement)
	if err != nil {
		return err
	}
	*t = Transition(trw)
	t.set = true
	return nil
}

// Validate - validates the "Expiration" element
func (t Transition) Validate() error {
	if !t.set {
		return nil
	}

	if t.IsDaysNull() && t.IsDateNull() {
		return errXMLNotWellFormed
	}

	// Both transition days and date are specified
	if !t.IsDaysNull() && !t.IsDateNull() {
		return errTransitionInvalid
	}
	if t.StorageClass == "" {
		return errXMLNotWellFormed
	}
	return nil
}

// IsDaysNull returns true if days field is null
func (t Transition) IsDaysNull() bool {
	return t.Days == TransitionDays(0)
}

// IsDateNull returns true if date field is null
func (t Transition) IsDateNull() bool {
	return t.Date.Time.IsZero()
}

// IsNull returns true if both date and days fields are null
func (t Transition) IsNull() bool {
	return t.IsDaysNull() && t.IsDateNull()
}

/*
 * Copyright 2018-2020 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package logger

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/buildpack/libbuildpack/logger"
	"github.com/fatih/color"
)

const (
	BodyIndent   = "    "
	HeaderIndent = "  "

	indent = "      "
)

var (
	description      = color.New(color.FgBlue)
	error            = color.New(color.FgRed, color.Bold)
	errorName        = color.New(color.FgRed, color.Bold)
	errorDescription = color.New(color.FgRed)
	lines            = regexp.MustCompile(`(?m)^`)
	name             = color.New(color.FgBlue, color.Bold)
	warning          = color.New(color.FgYellow, color.Bold)

	errorEyeCatcher     string
	firstLineEyeCatcher string
	warningEyeCatcher   string
)

func init() {
	color.NoColor = false
	errorEyeCatcher = error.Sprint("----->")
	firstLineEyeCatcher = color.New(color.FgRed, color.Bold).Sprint("----->")
	warningEyeCatcher = warning.Sprint("----->")
}

// Logger is an extension to libbuildpack.Logger to add additional functionality.
type Logger struct {
	logger.Logger
}

// Title prints the buildpack description flush left, with an empty line above it.
func (l Logger) Title(v Identifiable) {
	if !l.IsInfoEnabled() {
		return
	}

	l.Info("\n%s", l.prettyIdentity(v, name, description))
}

// Terminal error prints the build description colored red and bold, flush left, with an empty line above it, followed
// by the log message message red and bold, and indented two spaces.
func (l Logger) TerminalError(v Identifiable, format string, args ...interface{}) {
	if !l.IsInfoEnabled() {
		return
	}

	l.Info("\n%s", l.prettyIdentity(v, errorName, errorDescription))
	l.HeaderError(format, args...)
}

// Header prints the log message indented two spaces, with an empty line above it.
func (l Logger) Header(format string, args ...interface{}) {
	if !l.IsInfoEnabled() {
		return
	}

	l.Info("%s%s", HeaderIndent, fmt.Sprintf(format, args...))
}

// HeaderError prints the log message colored red and bold, indented two spaces, with an empty line above it.
func (l Logger) HeaderError(format string, args ...interface{}) {
	if !l.IsInfoEnabled() {
		return
	}

	l.Header(error.Sprintf(format, args...))
}

// HeaderWarning prints the log message colored yellow and bold, indented two spaces, with an empty line above it.
func (l Logger) HeaderWarning(format string, args ...interface{}) {
	if !l.IsInfoEnabled() {
		return
	}

	l.Header(warning.Sprintf(format, args...))
}

// Body prints the log message with each line indented four spaces.
func (l Logger) Body(format string, args ...interface{}) {
	if !l.IsInfoEnabled() {
		return
	}

	l.Info(color.New(color.Faint).Sprint(
		strings.ReplaceAll(
			l.BodyIndent(format, args...),
			fmt.Sprintf("\x1b[%dm", color.Reset),
			fmt.Sprintf("\x1b[%dm\x1b[%dm", color.Reset, color.Faint))))
}

// BodyError prints the log message colored red and bold with each line indented four spaces.
func (l Logger) BodyError(format string, args ...interface{}) {
	if !l.IsInfoEnabled() {
		return
	}

	l.Body(error.Sprintf(format, args...))
}

// BodyIndent indents each line of a log message to the BodyIndent offset.
func (l Logger) BodyIndent(format string, args ...interface{}) string {
	return lines.ReplaceAllString(fmt.Sprintf(format, args...), BodyIndent)
}

// BodyWarning prints the log message colored yellow and bold with each line indented four spaces.
func (l Logger) BodyWarning(format string, args ...interface{}) {
	if !l.IsInfoEnabled() {
		return
	}

	l.Body(warning.Sprintf(format, args...))
}

func (l Logger) prettyIdentity(v Identifiable, nameColor *color.Color, descriptionColor *color.Color) string {
	if v == nil {
		return ""
	}

	name, description := v.Identity()

	if description == "" {
		return nameColor.Sprint(name)
	}

	return fmt.Sprintf("%s %s", nameColor.Sprint(name), descriptionColor.Sprint(description))
}

// PrettyIdentity formats a standard pretty identity of a type.
//
// Deprecated: Use Title
func (l Logger) PrettyIdentity(v Identifiable) string {
	if v == nil {
		return ""
	}

	n, d := v.Identity()

	if d == "" {
		return name.Sprint(n)
	}

	return fmt.Sprintf("%s %s", name.Sprint(n), description.Sprint(description))
}

// Error prints the log message with the error eye catcher.
//
// Deprecated: Use HeaderError or BodyError
func (l Logger) Error(format string, args ...interface{}) {
	if !l.IsInfoEnabled() {
		return
	}

	l.Info("%s %s", errorEyeCatcher, fmt.Sprintf(format, args...))
}

// FirstLine prints the log messages with the first line eye catcher.
//
// Deprecated: Use Title
func (l Logger) FirstLine(format string, args ...interface{}) {
	if !l.IsInfoEnabled() {
		return
	}

	l.Info("%s %s", firstLineEyeCatcher, fmt.Sprintf(format, args...))
}

// SubsequentLine prints log message with the subsequent line indent.
//
// Deprecated: Use Body
func (l Logger) SubsequentLine(format string, args ...interface{}) {
	if !l.IsInfoEnabled() {
		return
	}

	l.Info("%s %s", indent, fmt.Sprintf(format, args...))
}

// Warning prints the log message with the warning eye catcher.
//
// Deprecated: Use HeaderWarning or BodyWarning
func (l Logger) Warning(format string, args ...interface{}) {
	if !l.IsInfoEnabled() {
		return
	}

	l.Info("%s %s", warningEyeCatcher, fmt.Sprintf(format, args...))
}

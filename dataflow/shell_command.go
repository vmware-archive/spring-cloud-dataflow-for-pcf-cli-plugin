/*
 * Copyright (C) 2017-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under
 * the terms of the under the Apache License, Version 2.0 (the "License”);
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
package dataflow

import "os/exec"

func DataflowShellCommand(fileName string, dataflowServerUrl string, skipSslValidation bool) *exec.Cmd {
	cmd := exec.Command("java", "-jar", fileName, "--dataflow.uri="+dataflowServerUrl,
		"--dataflow.credentials-provider-command=cf oauth-token")
	if skipSslValidation {
		cmd.Args = append(cmd.Args, "--dataflow.skip-ssl-validation=true")
	}
	return cmd
}

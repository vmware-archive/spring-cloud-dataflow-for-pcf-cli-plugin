/*
 * Copyright (C) 2018-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under
 * the terms of the under the Apache License, Version 2.0 (the "License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package skipper

import "os/exec"

func SkipperShellCommand(fileName string, skipperServerUrl string, skipSslValidation bool) *exec.Cmd {
	cmd := exec.Command("java", "-jar", fileName, "--spring.cloud.skipper.client.serverUri="+skipperServerUrl,
		"--spring.cloud.skipper.client.credentials-provider-command=cf oauth-token")
	if skipSslValidation {
		cmd.Args = append(cmd.Args, "--spring.cloud.skipper.client.skip-ssl-validation=true")
	}
	return cmd
}

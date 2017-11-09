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
package shell

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

func RunShell(cmd *exec.Cmd) error {
	cmd.Env = envVars("PATH", "HOME", "CF_HOME")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Printf("Error accessing shell's standard input pipe: %s\n", err)
		return err
	}
	defer stdin.Close()

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go func() {
		io.Copy(stdin, os.Stdin)
	}()

	return cmd.Run()
}

func envVars(keys ...string) []string {
	vars := []string{}
	for _, key := range keys {
		if val, set := os.LookupEnv(key); set {
			vars = append(vars, fmt.Sprintf("%s=%s", key, val))
		}
	}
	return vars
}

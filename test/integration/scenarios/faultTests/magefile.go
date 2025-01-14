//go:build mage

/*
 * Copyright (c) Microsoft Corporation.
 * Licensed under the MIT license.
 * SPDX-License-Identifier: MIT
 */

package main

import (
	"fmt"
	"os"

	"github.com/eclipse-symphony/symphony/test/integration/lib/testhelpers"
	"github.com/eclipse-symphony/symphony/test/integration/scenarios/faultTests/utils"
	"github.com/princjef/mageutil/shellcmd"
)

// Entry point for running the tests
func FaultTests() error {
	fmt.Println("Running fault injection tests")

	// Run fault injection tests
	for _, test := range utils.Faults {
		err := FaultTestHelper(test)
		if err != nil {
			return err
		}
	}
	return nil
}

func FaultTestHelper(test utils.FaultTestCase) error {
	testName := fmt.Sprintf("%s/%s/%s", test.TestCase, test.Fault, test.FaultType)
	fmt.Println("Running ", testName)

	// Step 2.1: setup cluster
	defer testhelpers.Cleanup(testName)
	err := testhelpers.SetupCluster()
	if err != nil {
		return err
	}
	// Step 2.2: enable port forward on specific pod
	stopChan := make(chan struct{}, 1)
	defer close(stopChan)
	err = testhelpers.EnablePortForward(test.PodLabel, utils.LocalPortForward, stopChan)
	if err != nil {
		return err
	}

	InjectCommand := fmt.Sprintf("curl localhost:%s/%s -XPUT -d'%s'", utils.LocalPortForward, test.Fault, test.FaultType)
	os.Setenv(utils.InjectFaultEnvKey, InjectCommand)
	os.Setenv(utils.PodEnvKey, test.PodLabel)

	err = Verify(test.TestCase)
	return err
}

func Verify(test string) error {
	err := shellcmd.Command("go clean -testcache").Run()
	if err != nil {
		return err
	}
	err = shellcmd.Command(fmt.Sprintf("go test -v -timeout %s %s", utils.TEST_TIMEOUT, test)).Run()
	if err != nil {
		return err
	}

	return nil
}

// Copyright © 2016 Sidharth Kshatriya
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package engine

import "fmt"

func handleStepInto(es *engineState, dCmd dbgpCmd) string {
	gotoMasterBpLocation(es, dCmd.reverse)

	filename := xSlashSgdb(es.gdbSession, "filename")
	lineno := xSlashDgdb(es.gdbSession, "lineno")
	return fmt.Sprintf(gStepIntoBreakXMLResponseFormat, dCmd.seqNum, filename, lineno)
}

func handleStepOverOrOut(es *engineState, dCmd dbgpCmd, stepOut bool) string {
	command := "step_over"
	if stepOut {
		command = "step_out"
	}

	currentPhpStackLevel := xSlashDgdb(es.gdbSession, "level")
	levelLimit := currentPhpStackLevel
	if stepOut && currentPhpStackLevel > 0 {
		levelLimit = currentPhpStackLevel - 1
	}

	// We're interested in maintaining or decreasing the stack level for step over
	// We're interested in strictly decreasing the stack level for step out
	id := setPhpStackDepthLevelBreakpointInGdb(es, levelLimit)
	_, ok := continueExecution(es, dCmd.reverse)

	if !dCmd.reverse {
		// Cleanup
		removeGdbBreakpoint(es, id)

		gotoMasterBpLocation(es, false)
	} else {
		// A user (php) breakpoint was hit
		if ok {
			// Cleanup
			removeGdbBreakpoint(es, id)

			// What stack level are we on currently?
			levelLimit := xSlashDgdb(es.gdbSession, "level")

			// Disable all currently active breaks
			bpList := getEnabledPhpBreakpoints(es)
			disableGdbBreakpoints(es, bpList)

			// Step over/out in reverse to the previous statement with all other breaks disabled
			id2 := setPhpStackDepthLevelBreakpointInGdb(es, levelLimit)
			continueExecution(es, true)

			// Remove this one too
			removeGdbBreakpoint(es, id2)

			enableGdbBreakpoints(es, bpList)

		} else {
			// Disable all currently active breaks
			bpList := getEnabledPhpBreakpoints(es)
			disableGdbBreakpoints(es, bpList)

			// Do this again with the php stack level breakpoint enabled
			continueExecution(es, true)
			enableGdbBreakpoints(es, bpList)

			// Cleanup
			removeGdbBreakpoint(es, id)
		}

		// Note that we run in forward direction, even though we're in reverse mode
		gotoMasterBpLocation(es, false)
	}

	filename := xSlashSgdb(es.gdbSession, "filename")
	phpLineno := xSlashDgdb(es.gdbSession, "lineno")

	return fmt.Sprintf(gRunOrStepBreakXMLResponseFormat, command, dCmd.seqNum, filename, phpLineno)
}

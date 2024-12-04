package task

import (
	"pulsyflux/sliceext"
	"runtime"
	"strings"
)

func getCallstack() sliceext.Stack[string] {
	clStk := sliceext.NewStack[string]()
	frame := 0
	funcName := getFrame(frame).Function
	for funcName != "unknown" {
		if !strings.Contains(funcName, "task.getCallstack") {
			clStk.Push(funcName)
		}
		frame += 1
		funcName = getFrame(frame).Function
	}
	return clStk
}

func getFrame(skipFrames int) runtime.Frame {
	// We need the frame at index skipFrames+2, since we never want runtime.Callers and getFrame
	targetFrameIndex := skipFrames + 2

	// Set size to targetFrameIndex+2 to ensure we have room for one more caller than we need
	programCounters := make([]uintptr, targetFrameIndex+2)
	n := runtime.Callers(0, programCounters)

	frame := runtime.Frame{Function: "unknown"}
	if n > 0 {
		frames := runtime.CallersFrames(programCounters[:n])
		for more, frameIndex := true, 0; more && frameIndex <= targetFrameIndex; frameIndex++ {
			var frameCandidate runtime.Frame
			frameCandidate, more = frames.Next()
			if frameIndex == targetFrameIndex {
				frame = frameCandidate
			}
		}
	}

	return frame
}

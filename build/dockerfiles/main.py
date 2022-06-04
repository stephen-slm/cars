import json
import os
import subprocess
import sys
import time

compile_dict = json.loads(sys.argv[1])

startCompile = time.time_ns()

if compile_dict["compileSteps"] is not None:
    for compileStep in compile_dict["compileSteps"]:
        compileStep = compileStep.replace("{{sourceFile}}", compile_dict["sourceFile"])
        compileStep = compileStep.replace("{{stdInFile}}", compile_dict["stdInFile"])

        subprocess.check_output(compileStep, shell=True)

endCompile = time.time_ns()
startRun = time.time_ns()

for runStep in compile_dict["runSteps"]:
    runStep = runStep.replace("{{sourceFile}}", compile_dict["sourceFile"])
    runStep = runStep.replace("{{stdInFile}}", compile_dict["stdInFile"])
    os.system(runStep)

endRun = time.time_ns()
print("*-COMPILE::EOF-*", (endRun - startRun), (endCompile - startCompile))

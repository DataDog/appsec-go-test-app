#!/bin/bash

vegeta attack -targets targets.vegeta -format http -duration=0 | tee vegeta.data

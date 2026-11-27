# Continuous Swarm Status

## Working
- Graph-clean detection and early stop
- Continuous-swarm command uses real SmartSteward
- Failed units requeued
- FailureSignal type for worker errors
- CollectFailureSignals helper

## Next
- Call CollectFailureSignals inside ContinuousRunner.Run and pass to Learning.ReviewSignals
- Make learned rules contain the failure detail so workers can act on them
- Scripted test that shows fewer failures after a learning round

The continuous path is real and improving.

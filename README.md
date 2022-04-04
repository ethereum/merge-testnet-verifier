# Merge Testnet Verifier

Performs a continuous verification of a Merge (Execution-Consensus) testnet and validates the final health by measuring and comparing a set of given metrics Post-Merge.

## Command Line Arguments

###### `--client`
Execution/Beacon client URL endpoint to check for the client's status in the form: 
<Client name>,http://<URL>:<IP>.
Parameter can appear multiple times for multiple clients.

###### `--ttd`
Terminal Total Difficulty of the Testnet.

###### `--override-verifications`
Specifies the path to a verifications YML file to override the default verifications.

###### `--extra-verifications`
Specifies the path to a verifications YML file to be added to the default verifications.

###### `--ttd-epoch-limit`
Max number of epochs to wait for the TTD to be reached.
Disable timeout: 0.
Default: 5

###### `--verif-epoch-limit`
Max number of epochs to wait for successful verifications after the merge has occurred.
Disable timeout: 0.
Default: 5

## Verifications YML File Format
The verifications file is a YML formatted file that contains a list of all verifications to be performed during the testnet's runtime.

Each verification element contains the following fields (all mandatory unless specified otherwise):
##### - VerificationName, string
Name of the verification, which will be printed on the output to help identify when the verification fails.
##### - ClientLayer, string
Layer from which the verification data will be obtained -- only accepts "Execution" or "Beacon" values thus far.
##### - PostMerge, bool
Whether the verification data should only be gathered after the merge has occurred (true) or during all the testnet's runtime (false).
##### - MetricName, string
Metric to be collected which then will be aggregated and compared to obtain the verification's outcome. Only one data point of this metric will be collected per block/slot. See Supported Metrics section.

##### - AggregateFunction, string
Aggregation function used to produce a single value that can be compared in the PassCriteria. See Supported Aggregate Functions section.
##### - AggregateFunctionValue, string, optional
Optional aggregate value used for some of the aggregation functions.
##### - PassCriteria, string
Comparison criteria used to determine a successful verification. See Supported Pass Criterias secion.
##### - PassValue, string
Pass value used in the PassCriteria comparison.

## Supported Metrics
### Execution Layer
##### - ExecutionBlockCount
Number of execution blocks produced.
##### - ExecutionBaseFee
BaseFee Value of the block header.
##### - ExecutionGasUsed
Total gas used of the block header.
##### - ExecutionDifficulty
Difficulty value of the block header.
##### - ExecutionMixHash
MixHash value of the block header.
##### - ExecutionUnclesHash
Uncles hash value of the block header.
##### - ExecutionNonce
Nonce value of the block header.
### Beacon Layer
##### - BeaconBlockCount
Number of beacon blocks produced -- can only be 0 or 1 per slot.
##### - FinalizedEpoch
Number of times the `finalized_epoch` value in the `finality_checkpoints` changes values; 1 if the `finalized_epoch` value changes, 0 if the value is the same as the previous slot.
##### - JustifiedEpoch
Number of times the `justified_epoch` value in the `finality_checkpoints` changes values; 1 if the `justified_epoch` value changes, 0 if the value is the same as the previous slot.
##### - EpochAttestationPerformance
Attestation performance throughout the Epoch. Currently can only be obtained if a Lighthouse client is provided, since it uses the `validator_inclusion` endpoint and it's calculated by getting the ratio between  `previous_epoch_head_attesting_gwei` and `previous_epoch_active_gwei`.
##### - EpochTargetAttestationPerformance
Target attestation performance throughout the Epoch. Currently can only be obtained if a Lighthouse client is providedm, since it uses the `validator_inclusion` endpoint and it's calculated by getting the ratio between  `previous_epoch_target_attesting_gwei` and `previous_epoch_active_gwei`.
##### - SyncParticipationCount
Sync participation per slot -- Set bit count of `sync_committee_bits`
##### - SyncParticipationPercentage
Sync participation percentage per slot -- Set bit count of `sync_committee_bits` divided by the `SYNC_COMMITTEE_SIZE` value of the spec.

## Supported Aggregate Functions

##### - CountEqual
(Requires AggregateFunctionValue): Count all the values equal to AggregateFunctionValue.
##### - CountUnequal
(Requires AggregateFunctionValue): Count all the values not equal to AggregateFunctionValue.
##### - Average
Arithmetic mean of all the data points obtained -- 0 when no data points were obtained.
##### - Sum
Sum of all the data points obtained.
##### - Min
Minimum value of all data points obtained.
##### - Max
Maximum value of all data points obtained.
##### - Percentage
Percentage (0 - 100) of all data points that are greater than zero.

## Supported Aggregate Functions

##### - MinimumValue
Minimum value that the aggregated value can have in order for the verification to be successful.
##### - MaximumValue
Maximum value that the aggregated value can have in order for the verification to be successful.

## Default Verifications
See `default_verifications.yml`
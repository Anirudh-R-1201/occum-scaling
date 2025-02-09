import sys
import math
import json
from json import JSONDecodeError
from datetime import datetime, timedelta
from dataclasses import dataclass
from typing import List, Optional
from dataclasses_json import dataclass_json, LetterCase

@dataclass_json(letter_case=LetterCase.CAMEL)
@dataclass
class TimestampedReplica:
    """
    JSON data representation of a timestamped evaluation
    """
    time: str
    replicas: int

@dataclass_json(letter_case=LetterCase.CAMEL)
@dataclass
class AlgorithmInput:
    """
    JSON data representation of the algorithm input data
    """
    look_ahead: int
    current_replicas: int
    historical_replicas: List[TimestampedReplica]
    current_time: Optional[str] = None
    max_replicas: Optional[int] = None

def get_average_replicas_at_time(replica_data: List[TimestampedReplica], target_time: datetime) -> float:
    """
    Calculate average replicas for a specific time of day from historical data
    """
    matching_replicas = []
    target_minutes = target_time.hour * 60 + target_time.minute

    for data in replica_data:
        data_time = datetime.strptime(data.time, "%Y-%m-%dT%H:%M:%SZ")
        data_minutes = data_time.hour * 60 + data_time.minute
        if data_minutes == target_minutes:
            matching_replicas.append(data.replicas)

    return sum(matching_replicas) / len(matching_replicas) if matching_replicas else 1.0

def main():
    stdin = sys.stdin.read()

    if not stdin:
        print("No standard input provided to prediction adjustment algorithm, exiting", file=sys.stderr)
        sys.exit(1)

    try:
        algorithm_input = AlgorithmInput.from_json(stdin)
    except JSONDecodeError as ex:
        print(f"Invalid JSON provided: {str(ex)}, exiting", file=sys.stderr)
        sys.exit(1)
    except KeyError as ex:
        print(f"Invalid JSON provided: missing {str(ex)}, exiting", file=sys.stderr)
        sys.exit(1)

    current_time = datetime.utcnow()
    if algorithm_input.current_time:
        try:
            current_time = datetime.strptime(algorithm_input.current_time, "%Y-%m-%dT%H:%M:%SZ")
        except ValueError as ex:
            print(f"Invalid datetime format: {str(ex)}", file=sys.stderr)
            sys.exit(1)

    future_time = current_time + timedelta(minutes=algorithm_input.look_ahead)

    avg_current_replicas = get_average_replicas_at_time(algorithm_input.historical_replicas, current_time)
    avg_future_replicas = get_average_replicas_at_time(algorithm_input.historical_replicas, future_time)
 
    if avg_current_replicas > 0:
        prediction = (algorithm_input.current_replicas / avg_current_replicas) * avg_future_replicas
        weight_current, weight_historical = 0.3, 0.7   
        algorithm_input.current_replicas = (weight_current * algorithm_input.current_replicas + 
                                          weight_historical * avg_current_replicas)
    else:
        prediction = algorithm_input.current_replicas   
        algorithm_input.current_replicas = avg_current_replicas

    # Cap prediction at max_replicas if specified
    if algorithm_input.max_replicas is not None:
        prediction = min(prediction, algorithm_input.max_replicas)

    # Update the base JSON with the new current_replicas value
    base_json = json.loads(stdin)
    base_json["currentReplicas"] = algorithm_input.current_replicas
    json.dumps(base_json)

    print(math.ceil(prediction), end="")

if __name__ == "__main__":
    main()

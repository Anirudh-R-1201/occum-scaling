##TO DO convert traffic to replica count??

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
class TrafficData:
    """
    JSON data representation of traffic measurement at a point in time
    """
    time: str
    traffic: float #traffic snapshot

@dataclass_json(letter_case=LetterCase.CAMEL)
@dataclass
class AlgorithmInput:
    """
    JSON data representation of the algorithm input data
    """
    look_ahead: int
    current_traffic: float
    historical_traffic: List[TrafficData]
    current_time: Optional[str] = None
    traffic_per_replica: float

def get_average_traffic_at_time(traffic_data: List[TrafficData], target_time: datetime) -> float:
    """
    Calculate average traffic for a specific time of day from historical data
    """
    matching_traffic = []
    target_minutes = target_time.hour * 60 + target_time.minute

    for data in traffic_data:
        data_time = datetime.strptime(data.time, "%Y-%m-%dT%H:%M:%SZ")
        data_minutes = data_time.hour * 60 + data_time.minute
        if data_minutes == target_minutes:
            matching_traffic.append(data.traffic)

    return sum(matching_traffic) / len(matching_traffic) if matching_traffic else 1.0

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

    avg_current_traffic = get_average_traffic_at_time(algorithm_input.historical_traffic, current_time)
    avg_future_traffic = get_average_traffic_at_time(algorithm_input.historical_traffic, future_time)
 
    if avg_current_traffic > 0:
        prediction = (algorithm_input.current_traffic / avg_current_traffic) * avg_future_traffic
        weight_current, weight_historical = 0.3, 0.7   
        algorithm_input.current_traffic = (weight_current * algorithm_input.current_traffic + 
                                                weight_historical * avg_current_traffic)
    else:
        prediction = algorithm_input.current_traffic   
        algorithm_input.current_traffic = avg_current_traffic

    _ = json.dumps(algorithm_input.to_dict())

    # Convert traffic prediction to replica count by dividing by traffic per replica
    replicas = prediction / algorithm_input.traffic_per_replica if algorithm_input.traffic_per_replica > 0 else prediction
    print(math.ceil(replicas), end="")

if __name__ == "__main__":
    main()

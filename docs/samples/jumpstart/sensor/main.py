import requests
import json
import time
from datetime import datetime, timezone
import random
import os

wait_time = os.getenv('WAIT_TIME', 15)

def get_current_time():
    return datetime.now(timezone.utc).isoformat()

def create_payload(typeA_orders, typeB_orders, typeC_orders, typeA_defects, typeB_defects, typeC_defects):
    return {
        "orders": {
            "typeA": typeA_orders,
            "typeB": typeB_orders,
            "typeC": typeC_orders,
        },
        "defects": {
            "typeA": typeA_defects,
            "typeB": typeB_defects,
            "typeC": typeC_defects,
        },
        "time": get_current_time()
    }

def post_data(url, payload):
    headers = {'Content-Type': 'application/json'}
    try:
        print(f"Sending data to the server {url}: {payload}")
        response = requests.post(url, json=payload, headers=headers)
        response.raise_for_status()  # Raise an HTTPError for bad responses
    except requests.exceptions.RequestException as e:
        print(f"Error posting data: {e}")
        return None
    return response

def generate_defects(num_orders, defect_rate):
    return [1 if random.random() < defect_rate else 0 for _ in range(num_orders)]

def main():
    typeA_orders, typeB_orders, typeC_orders = 0, 0, 0
    typeA_defects, typeB_defects, typeC_defects = 0, 0, 0
    defect_rate = 0.01
    order_increase_num = 10
    # Ensure 'API_ENDPOINT' is set in your environment variables
    url = os.getenv('API_ENDPOINT', "http://localhost:5000") + "/submitData"
    while True:
        old_typeA_orders, old_typeB_orders, old_typeC_orders = typeA_orders, typeB_orders, typeC_orders
        order_increase_rate = random.uniform(0.5, 1.0)
        typeA_orders += round(order_increase_num * order_increase_rate)
        order_increase_rate = random.uniform(0.5, 1.0)
        typeB_orders += round(order_increase_num * order_increase_rate)
        order_increase_rate = random.uniform(0.5, 1.0)
        typeC_orders += round(order_increase_num * order_increase_rate)

        # Generate defects based on the defect rate
        typeA_defects += sum(generate_defects(typeA_orders - old_typeA_orders, defect_rate))
        typeB_defects += sum(generate_defects(typeB_orders - old_typeB_orders, defect_rate))
        typeC_defects += sum(generate_defects(typeC_orders - old_typeC_orders, defect_rate))

        payload = create_payload(typeA_orders, typeB_orders, typeC_orders, typeA_defects, typeB_defects, typeC_defects)
        response = post_data(url, payload)
        if response == None:
            print("Can't get response from posting data")
        else:
            print(f"Posted data at {payload['time']}, response status: {response.status_code}")
        time.sleep(wait_time)  # Wait several seconds before sending the next request

if __name__ == "__main__":
    main()
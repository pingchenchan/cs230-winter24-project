import os, time, threading, random
from flask import Flask, jsonify
import json
from kafka import KafkaProducer

app = Flask(__name__)

@app.route('/')
def home():
    return '''
    <!DOCTYPE html>
    <html>
    <head>
        <title>Home Page</title>
        <script>
            async function startDataCollection1() {
                const response = await fetch('/start_collect1');
                const data = await response.json();
                alert(data.message); 
                updateProgress();
            }

            async function startDataCollection2() {
                const response = await fetch('/start_collect2');
                const data = await response.json();
                alert(data.message); 
                updateProgress();
            }

            async function startDataCollection3() {
                const response = await fetch('/start_collect3');
                const data = await response.json();
                alert(data.message); 
                updateProgress();
            }

            async function stopDataCollection() {
                const response = await fetch('/stop_collect');
                const data = await response.json();
                alert(data.message); 
            }

            function updateProgress() {
                fetch('/progress')
                    .then(response => response.json())
                    .then(data => {
                        document.getElementById('status').innerText = `Status: ${data.status}`;
                        document.getElementById('cpu_usage').innerText = `CPU Usage: ${data.cpu_usage}%`;
                        document.getElementById('memory_usage').innerText = `Memory Usage: ${data.memory_usage}%`;
                        document.getElementById('throughput').innerText = `Throughput: ${data.throughput} mbps`;
                        document.getElementById('health').innerText = `Health: ${data.health}`;
                        document.getElementById('message').innerText = data.message;
                    })
                    .catch(error => console.error('Error fetching data:', error));
            }

            function updateKafkaTopic() {
                fetch('/get_kafka_topic')
                    .then(response => response.json())
                    .then(data => {
                        document.getElementById('kafka_topic').innerText = `This is: ${data.kafka_topic}`;
                    })
                    .catch(error => console.error('Error fetching Kafka topic:', error));
            }

            document.addEventListener('DOMContentLoaded', () => {
                setInterval(updateProgress, 1000); // Update progress every 1 second
                updateKafkaTopic();
            });
        </script>
    </head>
    <body>
        <h1 id="kafka_topic">This is </h1>
        <button onclick="startDataCollection1()">High CPU Usage</button>
        <button onclick="startDataCollection3()">Low CPU Usage</button>
        <button onclick="stopDataCollection()">Stop Data Collection</button>
        <p id="status">Status: idle</p>
        <p id="cpu_usage">CPU Usage: 50%</p>
        <p id="memory_usage">Memory Usage: 0%</p>
        <p id="throughput">Throughput: 0 mbps</p>
        <p id="health">Health: unknown</p>
        <p id="message">Click start to collect data.</p>
    </body>
    </html>
    '''

# Kafka configuration
kafka_broker = 'kafka:9092' 
kafka_topic = os.getenv('KAFKA_TOPIC', 'server_test')

# Create a Kafka producer
producer = KafkaProducer(bootstrap_servers=[kafka_broker],value_serializer=lambda v: json.dumps(v).encode('utf-8'))

# Progress of data collection
progress = {"status": "idle", "cpu_usage": 0, "memory_usage": 0,"throughput": 0,"health": "healthy", "message": ""}
stop_collect_flag = True




def collect_and_send_data1():
    global progress
    cpu_usage = 0 
    while cpu_usage < 80 and not stop_collect_flag:
        if cpu_usage > 70:
            cpu_usage = random.randint(70, 75)
        else:
            cpu_usage += random.randint(1, 5)
        memory_usage= random.randint(10, 80)
        throughput = random.randint(100, 500)
        health = "healthy" 
        # if cpu_usage < 70 else "unhealthy"
        # Create data point
        data_point = {
            "cpu_usage": cpu_usage,
            "memory_usage": memory_usage,
            "throughput": throughput,
            "health_status": health
        }
        # Send data point to Kafka topic
        producer.send(kafka_topic, value=data_point)
        print(f"Sent to Kafka -> CPU usage: {cpu_usage}%, Memory usage: {memory_usage}%, Throughput: {throughput} mbps, Health: {health}")
        progress = {
            "status": "collecting",
            "cpu_usage": cpu_usage,
            "memory_usage": memory_usage,
            "throughput": throughput,
            "health": health,
            "message": f"Sent to Kafka -> CPU usage: {cpu_usage}%, Memory usage: {memory_usage}%, Throughput: {throughput} mbps, Health: {health}"
        }
        time.sleep(1)  
    if stop_collect_flag:
        progress["status"] = "stopping"
        progress["message"] = "Data collection stopped."
    else:
        progress["status"] = "complete"
        progress["message"] = "Data collection and sending complete!"

def collect_and_send_data2():
    global progress
    cpu_usage = 0
    while cpu_usage <35 and not stop_collect_flag:
        if cpu_usage > 20:
            cpu_usage = random.randint(20, 30)
        else:
            cpu_usage += random.randint(1, 5)
        memory_usage= random.randint(10, 80)
        throughput = random.randint(100, 500)
        health = "healthy" if cpu_usage < 70 else "unhealthy"
        # Create data point
        data_point = {
            "cpu_usage": cpu_usage,
            "memory_usage": memory_usage,
            "throughput": throughput,
            "health_status": health
        }
        # Send data point to Kafka topic
        producer.send(kafka_topic, value=data_point)
        print(f"Sent to Kafka -> CPU usage: {cpu_usage}%, Memory usage: {memory_usage}%, Throughput: {throughput} mbps, Health: {health}")
        progress = {
            "status": "collecting",
            "cpu_usage": cpu_usage,
            "memory_usage": memory_usage,
            "throughput": throughput,
            "health": health,
            "message": f"Sent to Kafka -> CPU usage: {cpu_usage}%, Memory usage: {memory_usage}%, Throughput: {throughput} mbps, Health: {health}"
        }
        time.sleep(1)
    if stop_collect_flag:
        progress["status"] = "stopping"
        progress["message"] = "Data collection stopped."
    else:
        progress["status"] = "complete"
        progress["message"] = "Data collection and sending complete!"

def collect_and_send_data3():
    global progress
    cpu_usage = 50
    while cpu_usage <51 and not stop_collect_flag:
        if cpu_usage > 10:
            cpu_usage -= random.randint(1,5)
        else:
            cpu_usage = random.randint(1, 10)
        memory_usage= random.randint(10, 80)
        throughput = random.randint(100, 500)
        health = "healthy" if cpu_usage < 70 else "unhealthy"
        # Create data point
        data_point = {
            "cpu_usage": cpu_usage,
            "memory_usage": memory_usage,
            "throughput": throughput,
            "health_status": health
        }
        # Send data point to Kafka topic
        producer.send(kafka_topic, value=data_point)
        print(f"Sent to Kafka -> CPU usage: {cpu_usage}%, Memory usage: {memory_usage}%, Throughput: {throughput} mbps, Health: {health}")
        progress = {
            "status": "collecting",
            "cpu_usage": cpu_usage,
            "memory_usage": memory_usage,
            "throughput": throughput,
            "health": health,
            "message": f"Sent to Kafka -> CPU usage: {cpu_usage}%, Memory usage: {memory_usage}%, Throughput: {throughput} mbps, Health: {health}"
        }
        time.sleep(1)
    if stop_collect_flag:
        progress["status"] = "stopping"
        progress["message"] = "Data collection stopped."
    else:
        progress["status"] = "complete"
        progress["message"] = "Data collection and sending complete!"

@app.route('/start_collect1')
def start_collect1():
    global progress, stop_collect_flag
    if stop_collect_flag:
        stop_collect_flag = False 
        progress = {"status": "starting", "cpu_usage": 0, "memory_usage": 0,"throughput": 0,"health": "unknown", "message": "Starting data collection..."}
        thread = threading.Thread(target=collect_and_send_data1)
        thread.start()
        return jsonify({"message": "Data collection started."})
    else:
        return jsonify({"message": "Please stop the current data collection first."})
@app.route('/start_collect2')
def start_collect2():
    global progress,stop_collect_flag
    if stop_collect_flag:
        stop_collect_flag = False 
        progress = {"status": "starting", "cpu_usage": 0, "memory_usage": 0,"throughput": 0,"health": "unknown", "message": "Starting data collection..."}
        thread = threading.Thread(target=collect_and_send_data2)
        thread.start()
        return jsonify({"message": "Data collection started."})
    else:
        return jsonify({"message": "Please stop the current data collection first."})

@app.route('/start_collect3')
def start_collect3():
    global progress,stop_collect_flag
    if stop_collect_flag:
        stop_collect_flag = False 
        progress = {"status": "starting", "cpu_usage": 0, "memory_usage": 0,"throughput": 0,"health": "unknown", "message": "Starting data collection..."}
        thread = threading.Thread(target=collect_and_send_data3)
        thread.start()
        return jsonify({"message": "Data collection started."})
    else:
        return jsonify({"message": "Please stop the current data collection first."})

@app.route('/stop_collect')
def stop_collect():
    global stop_collect_flag
    stop_collect_flag = True
    progress["status"] = "stopping"
    progress["message"] = "Stopping data collection..."
    return jsonify({"message": "Data collection stopped."})

@app.route('/get_kafka_topic')
def get_kafka_topic():
    return jsonify({"kafka_topic": kafka_topic})

@app.route('/progress')
def get_progress():
    return jsonify(progress)

@app.route('/health_check', methods=['GET'])
def health_check():
    # loadbalencer check health status
    if progress['health'] == 'healthy':
        return jsonify({'status': 'Healthy'}),200
    else:
        return jsonify({'status': 'Unhealthy'}),503
    

if __name__ == '__main__':
    app.run(debug=True, host='0.0.0.0',port=5001)

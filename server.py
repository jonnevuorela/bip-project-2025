import socket
import json
from ultralytics import YOLO
import cv2
from collections import Counter

model = YOLO("yolov8n.pt")

# Starte Server (IP 0.0.0.0 erlaubt lokalen Zugriff im Netzwerk)
server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
server.bind(('0.0.0.0', 9999))
server.listen(1)
print("Waiting for connection...")

client_socket, address = server.accept()
print(f"Client connected from {address}")

cap = cv2.VideoCapture(0)

while True:
    ret, frame = cap.read()
    if not ret:
        break

    results = model(frame)
    object_list = []

    for result in results:
        for box in result.boxes:
            class_id = int(box.cls[0])
            class_name = result.names[class_id]
            object_list.append(class_name)

    counts = Counter(object_list)
    message = json.dumps(counts)
    client_socket.send(message.encode('utf-8'))

    annotated_frame = results[0].plot()
    cv2.imshow("Server - YOLO Detection", annotated_frame)

    if cv2.waitKey(1) & 0xFF == ord('q'):
        break

cap.release()
client_socket.close()
server.close()

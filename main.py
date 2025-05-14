from ultralytics import YOLO
import cv2
from collections import Counter
from states import States

state = States.OK 

# 1. Analyse eines Bildes
def analyze_image(image_path):
    print("\n--- Static Image Analysis ---")

    
    model = YOLO("yolov8n.pt")

    
    results = model(image_path)

    
    results[0].show()
    results[0].save(filename="ausgabe.jpg")

    
    object_list = []
    for result in results:
        for box in result.boxes:
            class_id = int(box.cls[0])
            confidence = float(box.conf[0])
            class_name = result.names[class_id]
            print(f"Object: {class_name}, Confidence: {confidence:.2f}")
            object_list.append(class_name)

    
   # counts = Counter(object_list)
 
def update_sign(path):
    image = cv2.imread(path)
    window_name= 'image_window'
    try:
        cv2.destroyWindow(window_name)
    except:
        pass  # ignore if the window doesn't exist yet

    cv2.imshow(window_name, image)
    cv2.waitKey(5000)
    
    


# 2. Live Detection mit Webcam
def live_camera_detection():
    global state
    print("\n--- Starting Live Detection ---")

    model = YOLO("yolov8n.pt")
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

        
        messages = []
        # if counts["person"] >= 3:
        #     messages.append("Caution: Many pedestrians detected!")
        # if "traffic light" in counts:
        #     messages.append("Traffic light ahead - follow the signal.")
        # if counts["car"] >= 5:
        #     messages.append("Heavy traffic - Reduce speed!")
        
        path = "path"

        
        if "traffic light" in counts:
            state=States.TRAFFIC_LIGHTS
        elif counts["car"] >= 5:
            state=States.BUSY
        elif counts["person"] >= 3:
            state=States.PEDESTRIANS
        else:
            state=States.OK
    
        print("\nDetected objects:", counts)
        match state:
            case state.OK:
                print("OK")
                path="SignsMedia/50.jpg"
            case state.BUSY:
                print("BUSY")
                path="SignsMedia/30.jpg"
            case state.PEDESTRIANS:
                print("A lot of pedestrians")
                path="SignsMedia/30.jpg"
            case state.TRAFFIC_LIGHTS:
                print("Traffic light ahead.")
                path="SignsMedia/light.jpg"

        update_sign(path)
        
        annotated_frame = results[0].plot()
        y = 30
        for msg in messages:
            cv2.putText(annotated_frame, msg, (10, y), cv2.FONT_HERSHEY_SIMPLEX, 0.6, (0, 0, 255), 2)
            y += 25

        cv2.imshow("Smart Traffic Sign - Live Detection", annotated_frame)

        
        if cv2.waitKey(1) & 0xFF == ord('q'):
            break

    cap.release()
    cv2.destroyAllWindows()


# Hauptprogramm
if __name__ == "__main__":

    analyze_image("SignsMedia/strasse.jpg")

    
    live_camera_detection()

from flask import Flask, render_template, request, jsonify
import sqlite3
from dash_app import create_dash_by_orders, create_dash_by_products, create_dash_by_process, fetch_defects_and_orders_count

app = Flask(__name__, static_folder="templates")
complete_rate = 82.6

@app.route("/")
def index():
    defects, orders = fetch_defects_and_orders_count()
    notifications = get_notifications("v2")
    alarm_count = 0
    for notification in notifications:
        if notification["type"] == "alarm":
            alarm_count += 1
    data_show = {
        "defects_rate": round(defects / orders * 100, 2),
        "finished_orders": round(orders * complete_rate / 100),
        "queue_orders": orders - round(orders * complete_rate / 100),
        "notifications": notifications,
        "alarm_count": alarm_count,
        "version": "2.0"
    }
    return render_template("index.html", data_show=data_show)

@app.route("/submitData", methods=["POST"])
def submit_data():
    data = request.get_json()
    conn = sqlite3.connect('data.db')
    cursor = conn.cursor()
    
    cursor.execute(
        "CREATE TABLE IF NOT EXISTS order_data (time TEXT NOT NULL PRIMARY KEY, typeA_orders INTEGER, typeB_orders INTEGER, typeC_orders INTEGER, typeA_defects INTEGER, typeB_defects INTEGER, typeC_defects INTEGER)"
    )

    orders = data['orders']
    defects = data['defects']
    time = data['time']
    
    cursor.execute(
        "INSERT INTO order_data (time, typeA_orders, typeB_orders, typeC_orders, typeA_defects, typeB_defects, typeC_defects) VALUES (?, ?, ?, ?, ?, ?, ?)",
        (time, orders['typeA'], orders['typeB'], orders['typeC'], defects['typeA'], defects['typeB'], defects['typeC'])
    )
    
    conn.commit()
    conn.close()
    return jsonify({"message": "Data received"}), 201

@app.errorhandler(404)
def page_not_found(e):
    return render_template('404.html'), 404

def get_notifications(version):
    if version == "v1":
        return [
            {"message": "Production Line 2 is offline.", "type": "alarm", "color": "red"},
            {"message": "A new order has been submitted.", "type": "warning", "color": "orange"},
            {"message": "A new order has been submitted.", "type": "warning", "color": "orange"},
            {"message": "Order #1108 is ready for shipping.", "type": "info", "color": "green"},
        ]
    else:
        return [
            {"message": "Dashboard version is upgraded successfully.", "type": "info", "color": "green"},
        ]

# Initialize Dash apps
dash_app_orders = create_dash_by_orders(app, "orders")
dash_app_products = create_dash_by_products(app, "products")
dash_app_process = create_dash_by_process(app, "process", complete_rate)

if __name__ == '__main__':
    app.run(debug=True)
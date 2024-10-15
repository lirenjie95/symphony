from flask import Flask, render_template, request, jsonify
import sqlite3
from dash_app import create_dash_by_orders, create_dash_by_products, create_dash_by_process

app = Flask(__name__)

@app.route("/")
def index():
    return render_template("index.html")

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

# Initialize Dash apps
dash_app_orders = create_dash_by_orders(app, "orders")
dash_app_products = create_dash_by_products(app, "products")
dash_app_process = create_dash_by_process(app, "process", 82.6)

if __name__ == '__main__':
    app.run(debug=True)
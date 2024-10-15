import dash
from dash import dcc, html
from dash.dependencies import Input, Output, State
import sqlite3
import plotly.express as px

# Function to fetch data from the database
def fetch_orders_count_data():
    conn = sqlite3.connect('data.db')
    cursor = conn.cursor()
    cursor.execute("SELECT time, typeA_orders, typeB_orders, typeC_orders FROM order_data")
    rows = cursor.fetchall()
    conn.close()
    x_values = [row[0] for row in rows]
    y_values = [(row[1] + row[2] + row[3]) for row in rows]
    return x_values, y_values

def fetch_orders_by_type_data():
    conn = sqlite3.connect('data.db')
    cursor = conn.cursor()
    cursor.execute("SELECT typeA_orders, typeB_orders, typeC_orders FROM order_data ORDER BY time DESC LIMIT 1")
    row = cursor.fetchone()
    conn.close()
    if row:
        typeA_orders = row[0]
        typeB_orders = row[1]
        typeC_orders = row[2]
        return typeA_orders, typeB_orders, typeC_orders
    else:
        return None, None, None, None

def fetch_defects_and_orders_count():
    conn = sqlite3.connect('data.db')
    cursor = conn.cursor()
    cursor.execute("SELECT time, typeA_defects, typeB_defects, typeC_defects, typeA_orders, typeB_orders, typeC_orders FROM order_data ORDER BY time DESC LIMIT 1")
    row = cursor.fetchone()
    conn.close()
    if row:
        total_defects = row[1] + row[2] + row[3]
        total_orders = row[4] + row[5] + row[6]
        return total_defects, total_orders
    else:
        return None, None

def create_dash_by_orders(flask_app, name):
    dash_app = dash.Dash(__name__, server=flask_app, url_base_pathname="/" + name + "/")
    
    # Dash layout
    dash_app.layout = html.Div([
        dcc.Graph(
            id=name,
            config={"displayModeBar": False},  # Hide toolbar
            style={"width": "100%", "height": "100%"}  # adaptive size
        ),
        dcc.Interval(
            id='interval-component-orders',
            interval=60*1000,  # in milliseconds (1 minute)
            n_intervals=0
        )
    ], style={"width": "100%", "height": "100%", "display": "flex", "flex-direction": "column"})

    # Dash callback
    @dash_app.callback(
        Output(name, "figure"),
        [Input('interval-component-orders', 'n_intervals')]
    )
    def update_graph(n_intervals):
        # Fetch data from the database
        x_values, y_values = fetch_orders_count_data()
        fig = px.line(x=x_values, y=y_values, labels={'x': 'time', 'y': 'Orders Count'})
        fig.update_layout(
            showlegend=False,  # Hide legend
            title=None,  # Hide title
            xaxis_title=None,  # Remove x-axis title
            yaxis_title=None,  # Remove y-axis title
            autosize=True,
            height=180,
            margin=dict(l=20, r=20, t=40, b=40),
        )
        return fig

    return dash_app

def create_dash_by_products(flask_app, name):
    dash_app = dash.Dash(__name__, server=flask_app, url_base_pathname="/" + name + "/")
    
    # Dash layout
    dash_app.layout = html.Div([
        dcc.Graph(
            id=name,
            config={"displayModeBar": False},  # Hide toolbar
            style={"width": "100%", "height": "100%", "overflow": "hidden"}  # adaptive size and hide overflow
        ),
        dcc.Interval(
            id='interval-component-products',
            interval=60*1000,  # in milliseconds (1 minute)
            n_intervals=0
        )
    ], style={"width": "100%", "height": "100%", "display": "flex", "flex-direction": "column"})

    # Dash callback
    @dash_app.callback(
        Output(name, "figure"),
        [Input('interval-component-products', 'n_intervals')]
    )
    def update_graph(n_intervals):
        # Fetch data from the database
        typeA_orders, typeB_orders, typeC_orders = fetch_orders_by_type_data()
        
        # Check if data is fetched correctly
        if typeA_orders is None or typeB_orders is None or typeC_orders is None:
            return dash.no_update
        
        # Create bar chart
        fig = px.bar(
            x=[typeA_orders, typeB_orders, typeC_orders],
            y=['Type A', 'Type B', 'Type C'],
            orientation='h',  # Horizontal bar chart
            color=['Type A', 'Type B', 'Type C'],
            color_discrete_map={'Type A': 'purple', 'Type B': 'yellow', 'Type C': 'blue'}
        )
        fig.update_layout(
            showlegend=False,  # Hide legend
            title=None,  # Hide title
            margin=dict(l=20, r=20, t=20, b=20),
            autosize=True,
            height=180,
            xaxis_title=None,  # Remove x-axis title
            yaxis_title=None,  # Remove y-axis title
        )
        return fig

    return dash_app

def create_dash_by_process(flask_app, name, progress):
    dash_app = dash.Dash(__name__, server=flask_app, url_base_pathname="/" + name + "/")
    
    labels = ["Completed", "Remaining"]
    values = [progress, 100 - progress]
    
    # Create pie chart
    fig = px.pie(
        names=labels,
        values=values,
        hole=0.4,  # This creates the donut shape
        labels={'names': 'Order Status', 'values': 'Percentage'}
    )
    # Dash layout
    dash_app.layout = html.Div([
        dcc.Graph(
            id="progress-pie-chart",
            figure=fig,
        )
    ])
    return dash_app
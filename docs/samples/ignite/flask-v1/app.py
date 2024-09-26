from flask import Flask, render_template
import dash
from dash import dcc
from dash import html
from dash.dependencies import Input, Output, State
import sqlite3
import plotly.express as px

app = Flask(__name__)

@app.route('/')
def index():
    return render_template('index.html')

# Initialize Dash
dash_app_temperature = dash.Dash(__name__, server=app, url_base_pathname='/temperature/')

# Dash layout
dash_app_temperature.layout = html.Div([
    dcc.Graph(id='line-chart'),
    dcc.Input(id='input-box', type='text', placeholder='Enter here'),
    html.Button('Update Temperature', id='update-button'),
    html.Div(id='output-state')
])

# Function to fetch data from the database
def fetch_data():
    conn = sqlite3.connect('environment.db')
    cursor = conn.cursor()
    cursor.execute("SELECT x_value, y_value1, y_value2 FROM line_data")
    # y_value1 is temperature, y_value2 is humidity
    rows = cursor.fetchall()
    conn.close()
    x_values = [row[0] for row in rows]
    y_values1 = [row[1] for row in rows]
    return x_values, y_values1

# Dash callback
@dash_app_temperature.callback(
    Output('line-chart', 'figure'),
    [Input('update-button', 'n_clicks')],
    [State('input-box', 'value')]
)
def update_graph(n_clicks, value):
    # Fetch data from the database
    x_values, y_values = fetch_data()
    data = {
        'data': [
            {'x': x_values, 'y': y_values, 'type': 'line', 'name': 'temperature'},
        ],
        'layout': {'title': 'Temperature Data'}
    }
    return data

def create_dash_small(flask_app):
    dash_app_small = dash.Dash(
        server=flask_app,
        name="Dashboard",
        url_base_pathname='/temperature-small/'
    )
    x_values, y_values = fetch_data()
    fig = px.line(x=x_values, y=y_values, labels={'x': 'Time', 'y': 'Temperature'})
    dash_app_small.layout = html.Div([
        dcc.Graph(id='temperature-small-graph', figure=fig)
    ])
    return dash_app_small

# create thumbnail dashboard
dash_app_temperature_small = create_dash_small(app)

if __name__ == '__main__':
    app.run(debug=True)
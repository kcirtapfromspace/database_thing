import pandas as pd

data = pd.read_csv('data.csv')
print(data.head())

# Perform data cleaning or manipulation
data = data.dropna()
data = data[data['column_name'] != 'some_value']

# Perform data analysis
print(data.describe())

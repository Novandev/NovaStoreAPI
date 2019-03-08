# Proposal for Novastore API/ Command-line utility

As a working data scientist, having the ability to have a client upload their own data or having an automated way to store models, return statistics, or just spin up api services is usually a team effort that draws together a mirad of programming disciplines to accomplish.

With my background in backend development and data science, the Novastore API and Command line interface hopes to correct this by combining a Python, Golang, SQL and AWS for a scalable system to give users and teams the ability to have their data analyzed, and predicted upon in a fashion that is more developer friendly command line interface.

## Novastore API
This is the portion that the command line connects to that authenticates and runs predictions upon a users given CSV file. The results are returned in JSON Format for the stats API, or in Pickle/ HDF5 format. This in the future will also support a limited number of endpoints for which preditions can be ran on this data

- Assumes data will be cleaned in the CSV files
- Users can NOT directly interact with this api

## Novastore Command Line Utitlity

The user facing side of the system. This portion acceps user inputs in the form of commans and users login to see their current stored models/states improve their models

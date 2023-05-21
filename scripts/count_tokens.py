from pymongo import MongoClient

client = MongoClient()
db = client.split

token_collection = db.tokens

count = token_collection.find_one({})

print(count)

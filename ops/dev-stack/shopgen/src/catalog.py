"""Reference data for the synthetic shop: geography, catalog, channels, payment rails."""

import random

# code, traffic weight, tax rate, cities
COUNTRIES = [
    ("US", 40, 0.08, ["New York", "Los Angeles", "Chicago", "Austin", "Seattle", "Miami"]),
    ("GB", 12, 0.20, ["London", "Manchester", "Edinburgh"]),
    ("DE", 10, 0.19, ["Berlin", "Munich", "Hamburg"]),
    ("CA", 8, 0.13, ["Toronto", "Vancouver", "Montreal"]),
    ("FR", 8, 0.20, ["Paris", "Lyon", "Marseille"]),
    ("AU", 6, 0.10, ["Sydney", "Melbourne"]),
    ("NL", 5, 0.21, ["Amsterdam", "Rotterdam"]),
    ("ES", 5, 0.21, ["Madrid", "Barcelona"]),
    ("JP", 4, 0.10, ["Tokyo", "Osaka"]),
    ("BR", 2, 0.17, ["Sao Paulo", "Rio de Janeiro"]),
]
TAX_RATE = {c[0]: c[2] for c in COUNTRIES}

# name, weight, price range, gross margin, brands, product nouns
CATEGORIES = [
    ("Electronics", 18, (19, 349), 0.22, ["Voltix", "Auralis", "Nimbus"],
     ["Wireless Earbuds", "USB-C Charger", "Smart Speaker", "Fitness Tracker", "Power Bank", "Mechanical Keyboard"]),
    ("Home & Kitchen", 16, (12, 189), 0.38, ["Hearthly", "Copperleaf", "Nordvik"],
     ["Pour-Over Kettle", "Chef Knife", "Linen Throw", "Ceramic Planter", "Air Fryer", "Cast Iron Skillet"]),
    ("Apparel", 17, (15, 129), 0.55, ["Northfold", "Kestrel", "Loam"],
     ["Merino Tee", "Rain Shell", "Trail Socks", "Denim Jacket", "Fleece Hoodie", "Stretch Chinos"]),
    ("Beauty", 11, (8, 79), 0.62, ["Lumen", "Petal & Pine"],
     ["Vitamin C Serum", "Clay Mask", "Lip Balm Set", "SPF 50 Sunscreen", "Hair Oil"]),
    ("Sports & Outdoors", 12, (14, 259), 0.35, ["Summit", "Riverbend"],
     ["Yoga Mat", "Hydration Pack", "Camping Lantern", "Resistance Bands", "Trekking Poles"]),
    ("Books", 9, (9, 45), 0.30, ["Paperlane", "Inkwell Press"],
     ["Field Guide", "Weeknight Cookbook", "Sci-Fi Novel", "Design Handbook", "Kids Atlas"]),
    ("Toys & Games", 9, (10, 89), 0.45, ["Pipsqueak", "Brickworks"],
     ["1000pc Puzzle", "Building Set", "Strategy Board Game", "Plush Fox", "RC Car"]),
    ("Pet Supplies", 8, (6, 99), 0.40, ["Wagwell", "Whisker Co"],
     ["Dog Harness", "Cat Tree", "Chew Toy Pack", "Orthopedic Pet Bed", "Grain-Free Treats"]),
]
VARIANTS = ["", " Pro", " Mini", " XL", " Lite", " 2", " Max", " Travel", " Classic"]

# name, weight, conversion multiplier
CHANNELS = [("organic", 34, 1.0), ("paid_search", 24, 1.15), ("social", 16, 0.7),
            ("email", 10, 1.5), ("referral", 9, 1.2), ("affiliate", 7, 0.9)]
DEVICES = [("mobile", 57, 0.8), ("desktop", 35, 1.3), ("tablet", 8, 1.0)]

# method, weight, [(provider, weight)]
PAYMENT_METHODS = [
    ("card", 60, [("stripe", 60), ("adyen", 40)]),
    ("paypal", 17, [("paypal", 100)]),
    ("apple_pay", 12, [("stripe", 50), ("adyen", 50)]),
    ("klarna", 7, [("klarna", 100)]),
    ("bank_transfer", 4, [("adyen", 100)]),
]
DECLINE_REASONS = [("insufficient_funds", 40), ("do_not_honor", 25), ("expired_card", 15),
                   ("incorrect_cvc", 12), ("fraud_suspected", 8)]

# carrier, service level, (min_days, max_days), countries served (None = all)
CARRIERS = [
    ("UPS", "standard", (2, 5), None), ("UPS", "express", (1, 2), None),
    ("FedEx", "standard", (2, 6), None), ("USPS", "ground", (3, 7), {"US"}),
    ("DHL", "international", (4, 9), None),
]
REFUND_REASONS = [("damaged", 25), ("wrong_item", 15), ("not_as_described", 25),
                  ("changed_mind", 30), ("late_delivery", 5)]
# share of delivered orders that get refunded, by category
REFUND_RATE = {"Electronics": 0.07, "Home & Kitchen": 0.05, "Apparel": 0.11, "Beauty": 0.03,
               "Sports & Outdoors": 0.05, "Books": 0.02, "Toys & Games": 0.04, "Pet Supplies": 0.03}

FIRST_NAMES = ["Ava", "Liam", "Noah", "Emma", "Olivia", "Mateo", "Sofia", "Lucas", "Mia", "Ethan",
               "Amara", "Hiro", "Yuki", "Lena", "Jonas", "Chloe", "Leo", "Isla", "Arjun", "Priya",
               "Omar", "Zara", "Felix", "Nora", "Diego", "Elena", "Kai", "Maya", "Theo", "Ines"]
LAST_NAMES = ["Thompson", "Garcia", "Muller", "Martin", "Rossi", "Silva", "Tanaka", "Nguyen", "Smith",
              "Kowalski", "Johansson", "Dubois", "Patel", "Kim", "Okafor", "Novak", "Fischer", "Lopez",
              "Brown", "Wilson", "Moreau", "Sato", "Costa", "Jensen", "Clarke"]

# relative traffic by UTC hour and by weekday (Mon=0)
HOURLY = [0.35, 0.25, 0.2, 0.18, 0.2, 0.3, 0.45, 0.6, 0.75, 0.85, 0.9, 0.95,
          1.0, 1.0, 0.95, 0.95, 1.0, 1.1, 1.25, 1.4, 1.45, 1.3, 0.9, 0.55]
WEEKDAY = [0.95, 0.95, 1.0, 1.0, 1.05, 1.2, 1.15]


def weighted(rng, items, weight_index=1):
    return rng.choices(items, weights=[i[weight_index] for i in items], k=1)[0]


def build_products(rng: random.Random, count: int):
    """Catalog with Zipf-like popularity so a few products dominate traffic."""
    products, skus = [], set()
    pid = 1
    while len(products) < count:
        cat = weighted(rng, CATEGORIES)
        name, _, (lo, hi), margin, brands, nouns = cat
        brand, noun, variant = rng.choice(brands), rng.choice(nouns), rng.choice(VARIANTS)
        title = f"{brand} {noun}{variant}"
        if title in skus:
            continue
        skus.add(title)
        price = int(rng.uniform(lo, hi)) + 0.99
        cost = round(price * (1 - margin) * rng.uniform(0.9, 1.1), 2)
        products.append({
            "product_id": pid, "sku": f"{name[:3].upper()}-{pid:05d}", "name": title,
            "category": name, "brand": brand, "price": price, "unit_cost": cost,
        })
        pid += 1
    ranks = list(range(1, count + 1))
    rng.shuffle(ranks)
    for p, r in zip(products, ranks):
        p["popularity"] = 1 / r ** 1.1
    return products

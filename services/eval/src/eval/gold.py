GOLD = [
    {
        "id": "q1_desalination",
        "question": "Какие методы обессоливания воды подходят, если сульфаты и хлориды по 200–300 мг/л, а сухой остаток ≤1000 мг/дм³?",
        "intent": "technology_search",
        "min_facts": 2,
    },
    {
        "id": "q2_catholyte",
        "question": "Какая скорость циркуляции католита оптимальна при электроэкстракции никеля?",
        "intent": "technology_search",
        "min_facts": 4,
        "min_contradictions": 1,
    },
    {
        "id": "q3_pgm",
        "question": "Покажи эксперименты и публикации по распределению Au, Ag и МПГ между штейном и шлаком за последние 5 лет",
        "intent": "technology_search",
        "min_facts": 0,
    },
    {
        "id": "q4_geo_compare",
        "question": "Способы закачки шахтных вод в глубокие горизонты в России и за рубежом и их технико-экономические показатели",
        "intent": "technology_search",
        "min_facts": 0,
    },
    {
        "id": "q5_coldleach",
        "question": "Есть ли данные по кучному выщелачиванию никелевой руды в холодном климате?",
        "intent": "gap_analysis",
        "min_facts": 0,
        "min_gaps": 1,
    },
    {
        "id": "q6_experts",
        "question": "Кто работал с электроэкстракцией никеля и циркуляцией католита?",
        "intent": "expert_search",
        "min_experts": 1,
    },
]

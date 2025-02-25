тестовые записи можно отправить через for i in {1..10}; do
    curl -X POST \
      http://localhost:8080/submit \
      -H 'Content-Type: application/json' \
      -d "{
        \"period_start\": \"2024-12-01\",
        \"period_end\": \"2024-12-31\",
        \"period_key\": \"month\",
        \"indicator_to_mo_id\": 227373,
        \"indicator_to_mo_fact_id\": 0,
        \"value\": 1,
        \"fact_time\": \"2024-12-31\",
        \"is_plan\": 0,
        \"auth_user_id\": 40,
        \"comment\": \"buffer Test_$i\"
      }"
done
​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​​

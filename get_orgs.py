import requests
import json

def count_years_for_org(years, years_to_check):
    count = 0
    years_list = []
    for year in years_to_check:
        if year in years:
            count += 1
            years_list.append(year)
    return count, years_list

def sort_orgs_by_count_and_name(orgs_list):
    for i in range(len(orgs_list)):
        for j in range(len(orgs_list) - 1 - i):
            org1 = orgs_list[j]
            org2 = orgs_list[j + 1]
            
            should_swap = False
            if org1["count"] < org2["count"]:
                should_swap = True
            elif org1["count"] == org2["count"] and org1["name"] > org2["name"]:
                should_swap = True
            
            if should_swap:
                orgs_list[j], orgs_list[j + 1] = orgs_list[j + 1], orgs_list[j]
    
    return orgs_list

def get_gsoc_orgs():
    url = "https://api.gsocorganizations.dev/orgs/all"
    
    try:
        response = requests.get(url, timeout=10)
        response.raise_for_status()
        data = response.json()
    except requests.RequestException as e:
        print(f"Error fetching data: {e}")
        return
    
    org_counts = {}
    years_to_check = ["2022", "2023", "2024", "2025"]
    
    # Count how many times each org appeared
    for org_id, org_info in data.items():
        name = org_info.get("name", org_id)
        years = org_info.get("years", [])
        
        count, years_list = count_years_for_org(years, years_to_check)
        
        if count > 0:
            org_counts[org_id] = {
                "name": name,
                "count": count,
                "years": years_list,
                "url": org_info.get("url", "")
            }
    
    # Sort: first by count (descending), then by name (alphabetical)
    org_list = list(org_counts.values())
    sorted_orgs = sort_orgs_by_count_and_name(org_list)
    
    # Save to JSON
    try:
        with open("orgs.json", "w") as f:
            json.dump(sorted_orgs, f, indent=2)
        print(f"Saved {len(sorted_orgs)} organizations to orgs.json")
    except IOError as e:
        print(f"Error saving file: {e}")

if __name__ == "__main__":
    get_gsoc_orgs()
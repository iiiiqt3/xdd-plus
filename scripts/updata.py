import requests
import time
import re
import json
import sys

#获取青龙token   
def gettoken(host, client_id, client_secret):
    url = f"{host}/open/auth/token"
    headers = {'Content-Type': 'application/json'}
    params = {
        'client_id': client_id,
        'client_secret': client_secret
    }    
    try:
        response = requests.get(url, headers=headers, params=params)
        data = response.json()        
        token = data['data']['token']       
        return token        
    except Exception as e:
        print(e)
        return None

#获取指定环境变量备注的信息
def get_envs_reid(host, token, remark, env_name):
    url = f"{host}/open/envs?searchValue={env_name}"
    authorization = f"Bearer {token}"
    headers = {
        'Accept': 'application/json',
        'Authorization': authorization
    }
    
    try:
        response = requests.get(url, headers=headers)
        data = response.json() 
        
        for item in data["data"]:
            if item["remarks"] == remark:
                put_remarks = item["remarks"]
                put_name = item["name"]
                put_id = item["id"]
                put_value = item["value"]
                #print(put_remarks)
                #print(put_name)
                #print(put_id)
                #print(put_value)
                
                return put_remarks, put_name, put_id, put_value        
        
    except Exception as e:
        print(e)
        return None
    

#更新环境
def put_envs(host, token, get_ck, put_name, put_remarks, put_id):
    url = f"{host}/open/envs"
    authorization = f"Bearer {token}"
    headers = {
        'Accept': 'application/json',
        'Authorization': authorization
    }
    data = {
        "value": str(get_ck),
        "name": put_name,
        "remarks": put_remarks,
        "id": put_id
    }    

    response = requests.put(url, headers=headers, json=data)
    if response.status_code == 200:
        #print(response.json())
        print('更新成功')
    else:
        print(f"Error: {response.status_code}, {response.text}")


if __name__ == '__main__':
    host = 'http://ip:端口'    
    client_id = 'tDaaa_olaaa'    
    client_secret = 'v4Aaaa_ynaa-3dUaaaNoaaaa'
        
    get_ck = sys.argv[1]
    remark = sys.argv[2]
    env_name = sys.argv[3]

    # 获取青龙token
    token = gettoken(host, client_id, client_secret)

    # 获取指定环境变量备注的信息
    result = get_envs_reid(host, token, remark, env_name)

    # 检查是否成功获取到环境变量的信息
    if result is not None:
        put_remarks, put_name, put_id, put_value = result

        # 判断 get_ck 和 put_value 是否相同，如果相同则退出
        if str(get_ck) == str(put_value):
            print("更新失败，提交的ck和服务器原有ck相同。")
            sys.exit()

        # 进行更新环境变量的操作
        put_envs(host, token, get_ck, put_name, put_remarks, put_id)
    else:
        print(f"未找到备注为 {remark} 的环境变量。")

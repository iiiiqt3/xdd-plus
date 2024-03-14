import requests
import sys

class QingLongApi:
    def __init__(self) -> None:
        # 青龙面板地址、账号密码
        self.host = 'http://ip:端口'        
        self.client_id = 'tDaaa_olaaa'        
        self.client_secret = 'v4Aaaa_ynMi-3aaaycNo0aaa'        
        self.token = self.get_token()
        self.ck = None

    def get_token(self):
        try:
            url = self.host + f"/open/auth/token?client_id={self.client_id}&client_secret={self.client_secret}"
            data = requests.get(url).json()
            token = data['data']['token']
            return token
        except Exception as e:
            return None

    def get_remarks(self, env_name):
        headers = {
            'Authorization': 'Bearer ' + self.token,
            'Content-Type': 'application/json'
        }
        res = requests.get(self.host + f'/open/envs?searchValue={env_name}', headers=headers).json()
        number = len(res['data'])
        info = []
        for i in range(number):
            info.append(res['data'][i]['remarks'])
        return info

    def submit_env(self, env_name, value, remark):
        url = self.host + "/open/envs"
        payload = [{
            "name": env_name,
            "value": value,
            "remarks": remark,
        }]
        headers = {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer ' + self.token
        }
        response = requests.request("POST", url, headers=headers, json=payload).json()
        #print(response)
        if response.get('code') == 200:        
            print ('记录成功')
            return 200
        else:
            #return response.get('message')
            print ('记录失败，已有相同CK')


if __name__ == '__main__':
    api = QingLongApi()
    value = sys.argv[1]
    remark = sys.argv[2]
    env_name = sys.argv[3]
    remarks = api.get_remarks(env_name)
    #print(remarks)
    if remark in remarks:
        print("记录失败，已有相同备注，需换一个备注.")               
    else:
        result = api.submit_env(env_name, value, remark)
        #print(result)

# Extracted API signals

## URLs

- https://hash-game-admin.iyes.dev/auth/login
- https://hash-game-admin.iyes.dev/auth/login/security
- https://hash-game-admin=
- https://hash.iyes.de=
- https://hash.iyes.dev
- https://hash.iyes=

## curl samples

curl 'ht=
tps://hash-game-admin.iyes.dev/auth/login' 
   
 -H 'accept: application/=
json' 
   
 -H 'origin: https://hash.iyes.dev' 
   
 -H 'referer: http=
s://hash.iyes.dev/' 
   --data-raw '{"username":"testcq01","password":"=
testcq01","is_ai":true}' =

 =E7=A4=BA=E4=BE=8B=EF=BC=88=E9=82=AE=E7=AE=B1+=
=E9=AA=8C=E8=AF=81=E7=A0=81=EF=BC=8C=E9=9C=80=E5=85=88=E5=8F=91=E9=82=AE=E7=
=AE=B1=E9=AA=8C=E8=AF=81=E7=A0=81=EF=BC=89=EF=BC=9A 
 curl 'https:=
//hash-game-admin.iyes.dev/auth/login' 
   
 -H 'accept: application/json=
' 
   
 -H 'origin: https://hash.iyes.dev' 
   
 --data-raw '{"email":"=
user@example.com","email_code":"123456","is_ai":true}' 
 =E7=A4=BA=E4=BE=
=8B=EF=BC=88=E6=89=8B=E6=9C=BA+=E9=AA=8C=E8=AF=81=E7=A0=81=EF=BC=8C=E9=9C=
=80=E5=85=88=E5=8F=91=E6=89=8B=E6=9C=BA=E9=AA=8C=E8=AF=81=E7=A0=81=EF=BC=89=
=EF=BC=9A 
 curl 'https://hash-game-admin.iyes.dev/auth/login' 
   
 -H 'accept: application/json' 
   
 -H 'origin: https://hash.iyes.de=
v' 
   
 --data-raw '{"country_code":"86","phone":"13800138000","phone_co=
de":"123456","is_ai":true}' 
 2.3 =E9=A6=96=E6=AC=A1=E7=99=BB=E5=BD=95=E8=AE=BE=E7=
=BD=AE=E5=AF=86=E4=BF=9D =E6=8E=A5=E5=8F=
=A3=EF=BC=9APOST /auth/login/security 
 =E8=A7=A6=E5=8F=91=EF=BC=
=9A=E7=99=BB=E5=BD=95=E8=BF=94=E5=9B=9E code=3D40061 
 =E5=8F=82=
=E6=95=B0=EF=BC=9Ausername=E3=80=81password=E3=80=81new_password=E3=80=81wp=
_password=E3=80=81wp_password2=E3=80=81security_question=E3=80=81security_r=
eminder=E3=80=81security_code 
 =E5=AF=86=E4=BF=9D=E9=97=AE=E9=A2=
=98=E5=88=97=E8=A1=A8=EF=BC=9AGET /auth/security/questions 
 =E8=
=A7=84=E5=88=99=EF=BC=9A=E7=BC=96=E5=8F=B7 1=E3=80=818=E3=80=819=E3=80=8111=
=E3=80=8116 =E7=AD=94=E6=A1=88=E7=BA=AF=E6=95=B0=E5=AD=97=E4=B8=94=E4=B8=8D=
=E4=BD=8E=E4=BA=8E6=E4=BD=8D=EF=BC=9B=E5=85=B6=E4=BB=96=E7=BC=96=E5=8F=B7=
=E4=B8=AD=E6=96=87=E4=B8=94=E8=87=B3=E5=B0=912=E4=BD=8D=E3=80=82 
 curl 'https://hash-game-admin.iyes.dev/auth/login/security' 
   
 -H 'ac=
cept: application/json' 
   
 -H 'origin: https://hash.iyes.dev' 
   =
 
 =
 --data-raw '{"username":"testcq01","password":"testcq01","new_password":"1=
47258","wp_password":"147258","wp_password2":"147258","security_question":"=
1.=E6=82=A8=E7=9A=84=E5=AD=A6=E5=8F=B7(=E6=88=96=E5=B7=A5=E5=8F=B7)=E6=98=
=AF?","security_reminder":"147255","security_code":"147258"}' 
 2.4 =E8=B0=B7=E6=AD=
=8C/=E9=82=AE=E7=AE=B1/=E6=89=8B=E6=9C=BA=E4=BA=8C=E6=AC=A1=E8=AE=A4=E8=AF=
=81 =E8=A7=A6=E5=8F=91=EF=BC=9A=E7=99=BB=
=E5=BD=95=E8=BF=94=E5=9B=9E code=3D40045=EF=BC=8Cextra =E4=B8=AD=E5=B8=A6 k=
ey=EF=BC=88=E5=8D=B3 login_key=EF=BC=89=E3=80=81=E8=84=B1=E6=95=8F=E9=82=AE=
=E7=AE=B1/=E6=89=8B=E6=9C=BA=E7=AD=89=E3=80=82 
 =E7=AC=AC=E4=BA=
=8C=E6=AC=A1=E7=99=BB=E5=BD=95=E6=90=BA=E5=B8=A6 login_key=EF=BC=8C=E5=B9=
=B6=E4=BB=BB=E9=80=89 google_code =E6=88=96 email+email_code =E6=88=96 coun=
try_code+phone+phone_code=E3=80=82 
 curl 'https://hash-game-admin=
.iyes.dev/auth/login' 
   -H 'accept: application/json' 
   
 --data-=
raw '{"username":"testcq01","password":"testcq01","login_key":"=E4=B8=8A=E4=
=B8=80=E6=AD=A5=E8=BF=94=E5=9B=9E=E7=9A=84key","google_code":"763777","is_a=
i":true}' 
 2.5 =E8=B4=A6=E6=88=B7=E9=A3=8E=E9=99=A9=E9=87=8D=E7=BD=AE=E5=AF=86=E4=
=BF=9D =E6=8E=A5=E5=8F=A3=EF=BC=9APOST /au=
th/security/reset =

 =E8=A7=A6=E5=8F=91=EF=BC=9Acode=3D40034 =E6=88=
=96 40064 
 =E5=8F=82=E6=95=B0=EF=BC=9Ausername=E3=80=81password=
=E3=80=81security_key=E3=80=81=E5=8E=9F=E5=AF=86=E4=BF=9D=E3=80=81=E6=96=B0=
=E5=AF=86=E4=BF=9D=E3=80=81new_password=E3=80=81new_password2 =
 
 2.6 =E5=BC=82=
=E5=9C=B0=E7=99=BB=E5=BD=95 =E6=8E=A5=E5=
=8F=A3=EF=BC=9APOST /auth/except/city/login 
 =E8=A7=A6=E5=8F=91=
=EF=BC=9Acode=3D40069=EF=BC=8Cextra.logged_key=EF=BC=9B=E9=9C=80=E5=90=8E=
=E5=8F=B0=E5=BC=80=E5=90=AF=E5=BC=82=E5=9C=B0=E7=99=BB=E5=BD=95=E6=9D=83=E9=
=99=90 
 =E5=8F=82=E6=95=B0=EF=BC=9Akey(logged_key)=EF=BC=8C=E5=8F=
=8A google_code =E6=88=96 =E9=82=AE=E7=AE=B1/=E6=89=8B=E6=9C=BA=E9=AA=8C=E8=
=AF=81=E7=A0=81 =E6=88=96 =E5=AF=86=E4=BF=9D 
 2.7 =E5=88=B7=E6=96=B0 Token /=
 =E7=99=BB=E5=87=BA =E5=88=B7=E6=96=B0=EF=
=BC=9APOST /auth/refresh/token=EF=BC=8C=E5=8F=82=E6=95=B0 refresh_token=E3=
=80=81is_ai(=E5=8F=AF=E9=80=89) 
 =E7=99=BB=E5=87=BA=EF=BC=9AGET =
=E6=88=96 POST /auth/logout=EF=BC=8CHeader: Authorization: Bearer {token} 
 2=
.8 =E7=99=BB=E5=BD=95=E9=94=99=E8=AF=AF=E7=A0=81 40001 =E6=97=A0=E6=95=88=E7=99=BB=E5=BD=95=E5=90=8D 
 400=
02 =E5=AF=86=E7=A0=81=E9=94=99=E8=AF=AF 
 40036 =E5=AF=86=E7=A0=81=
=E8=BF=9E=E7=BB=AD=E9=94=99=E8=AF=AF=E9=94=81=E5=AE=9A 
 40045 =E9=
=9C=80=E4=BA=8C=E6=AC=A1=E8=AE=A4=E8=AF=81(=E8=B0=B7=E6=AD=8C/=E9=82=AE=E7=
=AE=B1/=E6=89=8B=E6=9C=BA) 
 40061 =E6=9C=AA=E8=AE=BE=E7=BD=AE=E5=
=AF=86=E4=BF=9D 40034/40064 =E8=B4=A6=E6=88=B7=E5=AE=89=E5=85=A8=
=E9=9A=90=E6=82=A3 
 40067 =E7=99=BB=E5=BD=95IP=E5=BC=82=E5=B8=B8=
=E8=81=94=E7=B3=BB=E5=AE=A2=E6=9C=8D 
 =
 40068 =E7=99=BB=E5=BD=95IP=
=E5=BC=82=E5=B8=B8=E8=81=94=E7=B3=BB=E4=B8=8A=E7=BA=A7 
 40069 =E5=
=BC=82=E5=9C=B0=E7=99=BB=E5=BD=95=E9=AA=8C=E8=AF=81 
 40070 =E5=BB=
=BA=E8=AE=AE=E4=BF=AE=E6=94=B9=E5=AE=89=E5=85=A8=E4=BF=A1=E6=81=AF 
 40073 login_key=E6=97=A0=E6=95=88 
 =
 40091 =E5=9B=BE=E5=BD=A2=E9=
=AA=8C=E8=AF=81=E7=A0=81=E5=BC=82=E5=B8=B8 
 40099 =E7=BB=BC=E5=90=
=88=E8=AE=A4=E8=AF=81=E5=A4=B1=E8=B4=A5 
 401/42001 =E6=9C=AA=E7=
=99=BB=E5=BD=95=E6=88=96token=E5=A4=B1=E6=95=88 
 =E5=90=8E=E7=BB=
=AD=E4=B8=9A=E5=8A=A1=E6=8E=A5=E5=8F=A3 Header =E6=90=BA=E5=B8=A6=EF=BC=9AA=
uthorization: Bearer {token} 
 3.=E7=94=A8=E6=88=B7=E4=BF=A1=E6=81=AF =
 3.1 =E7=94=A8=E6=88=B7=E5=9F=BA=E7=A1=80=E4=BF=A1=E6=81=AF=
=E5=92=8C=E4=BD=99=E9=A2=9D=E6=8E=A5=E5=8F=A3=EF=BC=9A =E6=8E=A5=E5=8F=A3=EF=BC=9AGET /api/users/i/info =
 
 He=
ader=EF=BC=9AAuthorization: Bearer {token} 
 =E8=BF=94=E5=9B=9E=E7=
=A4=BA=E4=BE=8B=EF=BC=88=E8=8A=82=E9=80=89=EF=BC=89=EF=BC=9A 
 {=
 
 "id": 757, 
 "username": "no8899", 
 "account": { 
 "b=
alance": 0.0, 
 "balance_trx": 0.0, 
 "balance_cny": 0.0, 
 =
 "balance_fixed": 0.0, =

 "balance_fixed_cny": 0.0 
 }, 
 "t=
wo_factor_auth": {"verified_google": true, "enabled_google": true} 
 } 
 3.2 =E7=
=94=A8=E6=88=B7=E8=B5=94=E7=8E=87=E8=BF=94=E7=82=B9=E4=BF=A1=E6=81=AF=E6=8E=
=A5=E5=8F=A3 =E6=8E=A5=E5=8F=A3=EF=BC=9AGE=
T /api/agents/i/real/rate 
 =E8=BF=94=E5=9B=9E=E5=AD=97=E6=AE=B5=
=EF=BC=9Areal_rate(=E5=AE=9E=E6=97=B6=E8=BF=94=E6=B0=B4)=E3=80=81lott_odds(=
=E5=BD=A9=E7=A5=A8=E8=B5=94=E7=8E=87)=E3=80=81slot_rate=E3=80=81live_rate=
=E3=80=81sport_rate =E7=AD=89 
 3.3 =E6=88=91=E7=9A=84=E5=AF=86=E4=BF=9D=E8=
=B5=84=E8=AE=AF=E6=8E=A5=E5=8F=A3=EF=BC=9A =E6=8E=A5=E5=8F=A3=EF=BC=9AGET /api/users/i/security_question 
 =
=E8=BF=94=E5=9B=9E=EF=BC=9Asecurity_question=E3=80=81security_reminder 
 3.4 =
=E6=88=91=E7=9A=84=E5=BD=A9=E7=A5=A8=E4=BB=8A=E6=97=A5=E6=8A=95=E6=B3=A8=E8=
=BE=93=E8=B5=A2=E6=83=85=E5=86=B5 =E6=8E=
=A5=E5=8F=A3=EF=BC=9AGET /api/web_bets/total 
 =E5=8F=82=E6=95=B0 =
filters=EF=BC=9Abet_time_dr =E4=B8=BA=E4=BB=8A=E6=97=A5=E6=97=A5=E6=9C=9F=
=E8=8C=83=E5=9B=B4=EF=BC=9Bgame_name_in =E4=BC=A0=E6=89=80=E6=9C=89=E5=BD=
=A9=E7=A5=A8=E5=90=8D=E7=A7=B0 =
 
 {"filters": { 
 "bet_time_d=
r": "2025-04-10~2025-04-11", 
 "game_name_in": ["=E5=93=88=E5=B8=8C=E4=
=B8=80=E5=88=86=E5=BD=A9", "=E5=93=88=E5=B8=8C=E4=B8=89=E5=88=86=E5=BD=A9",=
 "=E6=B3=A2=E5=9C=BA=E4=B8=80=E5=88=86=E5=BD=A9", "..."] 
 }} 
 5.=E5=BD=A9=
=E7=A7=8D=E5=BC=80=E5=A5=96=E5=8E=86=E5=8F=B2=E6=8E=A5=E5=8F=A3=EF=BC=9A 5.1 =E5=93=88=E5=B8=8C=E6=9E=81=E9=80=9F=E5=BD=
=A9 =EF=BC=88=E6=B3=A2=E5=9C=BA=
=E6=9E=81=E9=80=9F=E5=BF=AB=E4=B8=89=EF=BC=8C=E6=B3=A2=E9=95=BF=E6=9E=81=E9=
=80=9F=E8=B5=9B=E8=BD=A6 =E5=85=AC=E7=94=A8=
=E6=AD=A4=E5=8E=86=E5=8F=B2=EF=BC=89 
 =E6=8E=A5=E5=8F=A3=
=EF=BC=9A /api/lottery_log033s GET 
 =E6=89=80=E6=9C=89=E5=
=8E=86=E5=8F=B2=E6=8E=A5=E5=8F=A3=E5=8F=82=E8=80=83=E6=AD=A4=E8=BF=94=E5=9B=
=9E=E8=AF=B4=E6=98=8E =
 
 { 
 "data": [ 
 &nbs=
p; { 
 &=
nbsp;"id": 634552, &nbs=
p; "created": "2025-04-10T04:17:03+00:00", 
 =
; "block_time": "2025-04-10T04:17:03+00:=
00", # =E5=8C=BA=E5=9D=97=E6=97=B6=E9=97=B4=EF=BC=8C UTC=E6=97=B6=E9=97=B4 
 &n=
bsp; "block_num"=
: "71201600", # =E5=8C=BA=E5=9D=97=E9=AB=98=E5=BA=A6 
 =
 "block_hash": "00000000043e734041=
e380fc1cb6ac4c7c5cddeb011c5796a4b1d536b0569b73", 
 &nbs=
p; # =E5=8C=BA=E5=9D=97=E5=93=88=E5=B8=
=8C 
 &nbs=
p; "period=
s": "105202504101474", # =E6=9C=9F=E5=8F=B7 
 &nb=
sp; "last5_num": "56973", # =E6=9C=80=E5=
=90=8E5=E4=BD=8D=E6=95=B0=E5=AD=97=EF=BC=881=EF=BC=8C3=EF=BC=8C5=E5=88=86=
=E5=BD=A9=E6=9E=81=E9=80=9F=E5=BD=A9=E7=AD=89=E5=BC=80=E5=A5=96=E6=95=B0=E6=
=8D=AE=EF=BC=89 
 &=
nbsp;"last10_6_num": "15360", # =E6=9C=80=E5=90=8E10-6=E4=BD=8D=E6=95=B0=E5=
=AD=97=EF=BC=88=E6=96=B0=E4=BB=A5=E5=A4=AA=E5=9D=8A=E5=BC=80=E5=A5=96=E6=95=
=B0=E6=8D=AE=EF=BC=89 =

 &=
nbsp; "last11_5_num": "06,09,11,07,03", # 11x5=E7=8E=A9=E6=B3=95=
=E7=9A=84=E5=BC=80=E5=A5=96=E6=95=B0=E6=8D=AE 
 &=
nbsp; "last_pk10_num": "08,03,04,02,05,06,10,0=
1,09,07", # pk10=E7=8E=A9=E6=B3=95=E5=BC=80=E5=A5=96=E6=95=B0=E6=8D=AE 
 =
; "last_k3_num":=
 "4,2,2" # =E5=BF=AB=E4=B8=89=E7=8E=A9=E6=B3=95=E5=BC=80=E5=A5=96=E6=95=B0=
=E6=8D=AE 
 } 
 =E3=80=82=E3=80=82=E3=80=82 
 =
; ]} 
 5.2 =E5=93=88=E5=B8=8C=E4=B8=80=E5=88=86=E5=BD=A9=E6=9E=81=
=E9=80=9F =
 =EF=BC=88=E6=B3=A2=E5=
=9C=BA 1=E5=88=86=E5=BF=AB=E4=B8=89=EF=BC=8C=E6=B3=A2=E5=9C=BA11=E9=
=80=895 =E5=85=AC=E7=94=A8=E6=AD=A4=E5=8E=86=E5=8F=B2=EF=BC=89 
 =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/lottery_log103s?limit=3D20&page=3D1=
 
 5.3 =E5=93=88=E5=B8=8C=E4=B8=89=E5=88=86=E5=BD=A9=E6=9E=81=E9=80=9F =
 =
 =EF=BC=88=E6=B3=A2=E5=9C=BA 3=E5=
=88=86=E5=BF=AB=E4=B8=89=EF=BC=8C=E6=B3=A2=E5=9C=BA3=E5=88=8611=E9=80=895 =
=E5=85=AC=E7=94=A8=E6=AD=A4=E5=8E=86=E5=8F=B2=EF=BC=89 
 =E6=8E=
=A5=E5=8F=A3=EF=BC=9A /api/lottery_log303s?limit=3D20&page=3D1 =
 
 5.4 =
=E5=93=88=E5=B8=8C=E4=BA=94=E5=88=86=E5=BD=A9=E6=9E=81=E9=80=9F =
 =EF=BC=88=E6=B3=A2=E5=9C=BA 5=E5=88=86=
=E5=BF=AB=E4=B8=89=EF=BC=8C=E6=B3=A2=E5=9C=BA5=E5=88=8611=E9=80=895 =E5=85=
=AC=E7=94=A8=E6=AD=A4=E5=8E=86=E5=8F=B2=EF=BC=89 
 =E6=8E=A5=E5=
=8F=A3=EF=BC=9A /api/lottery_log503s?limit=3D20&page=3D1 
 5.5 =E6=B3=
=A2=E5=9C=BA=E6=9E=81=E9=80=9F=E5=BD=A9 =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/lottery_log05s?limit=3D20&pag=
e=3D1 
 5.6 =E6=B3=A2=E5=9C=BA=E4=B8=80=E5=88=86=E5=BD=A9 =
 =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/lottery_logs 
 =
5.7 =E6=B3=A2=E5=9C=BA=E4=B8=89=E5=88=86=E5=BD=A9 =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/lottery_log3s 
 5.8 =
=E6=B3=A2=E5=9C=BA=E4=BA=94=E5=88=86=E5=BD=A9 =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/lottery_log5s 
 5.9 =E4=
=BB=A5=E5=A4=AA=E5=9D=8A=E6=9E=81=E9=80=9F=E5=BD=A9 =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/eth_block_logs/?l=
imit=3D20&page=3D1 5.10 =E4=BB=A5=E5=A4=AA=E5=9D=8A1=E5=88=86=E5=BD=A9 =EF=BC=88=E4=BB=A5=E5=A4=AA=E6=9E=81=
=E9=80=9F=E8=B5=9B=E8=BD=A6=EF=BC=8C=E4=BB=A5=E5=A4=AA=E5=9D=8A=E5=BF=AB=E4=
=B8=89=EF=BC=8C=E4=BB=A5=E5=A4=AA=E5=9D=8A 11=E9=80=895=E5=85=AC=E7=
=94=A8=E6=AD=A4=E6=8E=A5=E5=8F=A3=EF=BC=89 
 =E6=8E=A5=E5=8F=A3=
=EF=BC=9A /api/eth_lottery_logs/?limit=3D20&page=3D1 
 5.11 =E4=BB=A5=
=E5=A4=AA=E5=9D=8A3=E5=88=86=E5=BD=A9 =EF=BC=88=E4=BB=A5=E5=A4=AA=E5=9D=8A 3=E5=88=86=E5=BF=AB=E4=B8=
=89=EF=BC=8C=E4=BB=A5=E5=A4=AA=E5=9D=8A3=E5=88=8611=E9=80=895=E5=85=AC=E7=
=94=A8=E6=AD=A4=E6=8E=A5=E5=8F=A3=EF=BC=89 
 =E6=8E=A5=E5=8F=A3=
=EF=BC=9A /api/eth_lottery_log3s/?limit=3D20&page=3D1 
 5.12 =E4=BB=A5=
=E5=A4=AA=E5=9D=8A5=E5=88=86=E5=BD=A9 =EF=BC=88=E4=BB=A5=E5=A4=AA=E5=9D=8A 5=E5=88=86=E5=BF=AB=E4=B8=
=89=EF=BC=8C=E4=BB=A5=E5=A4=AA=E5=9D=8A5=E5=88=8611=E9=80=895=E5=85=AC=E7=
=94=A8=E6=AD=A4=E6=8E=A5=E5=8F=A3=EF=BC=89 
 =E6=8E=A5=E5=8F=A3=
=EF=BC=9A /api/eth_lottery_log5s/?limit=3D20&page=3D1 
 5.13 =E5=B8=81=
=E5=AE=891=E5=88=86=E5=BD=A9 =EF=
=BC=88=E5=B8=81=E5=AE=89=E6=9E=81=E9=80=9F=E9=A3=9E=E8=89=87=EF=BC=8C=E5=B8=
=81=E5=AE=89 1=E5=88=86=E5=BF=AB=E4=B8=89=EF=BC=8C=E5=B8=81=E5=AE=891=
1=E9=80=895=E5=85=AC=E7=94=A8=E6=AD=A4=E6=8E=A5=E5=8F=A3=EF=BC=89 
 =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/bsc_lottery_logs/?limit=3D20&pag=
e=3D1 
 5.14 =E5=B8=81=E5=AE=893=E5=88=86=E5=BD=A9 =EF=BC=88=E5=B8=81=E5=AE=89 3=E5=88=86=E5=BF=AB=E4=
=B8=89=EF=BC=8C=E5=B8=81=E5=AE=893=E5=88=8611=E9=80=895=E5=85=AC=E7=94=A8=
=E6=AD=A4=E6=8E=A5=E5=8F=A3=EF=BC=89 
 =E6=8E=A5=E5=8F=A3=EF=BC=
=9A /api/bsc_lottery_log3s/?limit=3D20&page=3D1 
 5.15 =E5=B8=81=E5=AE=
=895=E5=88=86=E5=BD=A9 =EF=BC=88=
=E5=B8=81=E5=AE=89 5=E5=88=86=E9=A3=9E=E8=89=87=EF=BC=8C=E5=B8=81=E5=
=AE=895=E5=88=86=E5=BF=AB=E4=B8=89=EF=BC=8C=E5=B8=81=E5=AE=895=E5=88=8611=
=E9=80=895=E5=85=AC=E7=94=A8=E6=AD=A4=E6=8E=A5=E5=8F=A3=EF=BC=89 
 =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/bsc_lottery_log5s/?limit=3D20&pag=
e=3D1 
 5.16 =E5=8F=B0=E6=B9=BE5=E5=88=86=E5=BD=A9 =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/tw_lottery_logs/?l=
imit=3D30&page=3D1 =E8=BF=94=E5=9B=9E=E7=A4=BA=E4=BE=8B=EF=BC=
=9A 
 { 
 &nb=
sp; "data": [ 
 { 
 &nbs=
p; "id": 1783, 
 &nb=
sp; "created": "=
2025-04-10T04:35:00+00:00", 
 &=
nbsp; "block_time": "2025-04-10T04:35:00+00:00", 
 =
; "block_num": "114020164", =
# =E5=8C=BA=E5=9D=97=E9=AB=98=E5=BA=A6 
 &n=
bsp; "block_hash": "19,50,38,28,44,31,34,09,60,77,21=
,02,57,26,20,55,41,72,10,70", 
 =
; "periods": "114020164", # =E6=97=97=E5=8F=B7 
 &=
nbsp; "extra": {}, 
 =
; "last_tw5_num": "053=
97", # =E5=8F=B0=E6=B9=BE5=E7=B2=89=E5=BD=A9=E5=BC=80=E5=A5=96=E6=95=B0=E6=
=8D=AE 
 &=
nbsp; "las=
t_tw_pk10_num": "10,03,06,08,07,09,05,04,02,01", # =E5=8F=B0=E6=B9=BEpk10=
=E5=BC=80=E5=A5=96=E6=95=B0=E6=8D=AE 
 =
 &nbs=
p; "last_tw28_num": "5,8,2" # =E5=8F=B0=E6=B9=BE28=
=E5=BC=80=E5=A5=96=E6=95=B0=E6=8D=AE 
 =
 &nbs=
p;}, 
 5.17 =E7=A6=8F=E5=BD=A93D=E5=BC=80=E5=A5=96=E6=95=B0=E6=8D=AE =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/fc=
3d_lottery_logs/?limit=3D10&page=3D1 
 =E8=BF=94=E5=9B=9E=E7=A4=
=BA=E4=BE=8B=EF=BC=9A =
 
 { 
 "data": [ 
 &nbs=
p; { 
 &=
nbsp;"id": 11, 
 &n=
bsp;"created": "2025-04-09T13:10:00+00:00", 
 &nb=
sp; "block_time": "2025-04-09T13:10:00+00:00",=
 
 &=
nbsp; "block_num=
": "2025089", # =E5=8C=BA=E5=9D=97=E9=AB=98=E5=BA=A6 
 =
 "block_hash": "6,5,8", #=E5=
=8C=BA=E5=9D=97=E5=93=88=E5=B8=8C 
 &=
nbsp; "periods": "2025089", # =E6=97=97=E5=8F=B7 
 =
; "extra": { 
 =
; &n=
bsp;"next_period_time": "2025-04-10T13:10:00+00:00" # =E4=B8=8B=E6=9C=9F=E5=
=BC=80=E5=A7=8B=E6=97=B6=E9=97=B4 
 &=
nbsp; }, 
 &nb=
sp; "last_fc3d_num": "658" # =E7=A6=8F=E5=BD=A93D=E5=BC=80=E5=A5=
=96=E6=95=B0=E6=8D=AE =

 } 
 &=
nbsp; ]} 
 5.18 =E6=8E=92=E5=88=9735=E5=BC=80=E5=A5=96=
=E6=95=B0=E6=8D=AE =E6=8E=A5=E5=
=8F=A3=EF=BC=9A /api/pl35_lottery_logs/?limit=3D30&page=3D1 
 =E8=BF=94=E5=9B=9E=E7=A4=BA=E4=BE=8B=EF=BC=9A 
 { 
 "data":=
 [ 
 =
; { 
 &n=
bsp; "id": 11, 
 &nb=
sp; "created": "2025-04-09T13:10:00+00:00", 
 &nbs=
p; "block_time":=
 "2025-04-09T13:10:00+00:00", 
 =
; "block_num": "25089", # =E5=8C=BA=E5=9D=97=E9=AB=
=98=E5=BA=A6 
 &nbs=
p;"block_hash": "20883", # =E5=8C=BA=E5=9D=97=E5=93=88=E5=B8=8C 
 =
; "periods": "25=
089", # =E6=97=97=E5=8F=B7 
 &n=
bsp; "extra": { 
 &n=
bsp; "next_period_time": "2025-04-=
10T13:10:00+00:00" # =E4=B8=8B=E6=9C=9F=E5=BC=80=E5=A7=8B=E6=97=B6=E9=97=B4=
 
 &=
nbsp; }, 
 &nb=
sp; "last_pl35_num": "=
20883" # =E6=8E=92=E5=88=9735=E5=BC=80=E5=A5=96=E6=95=B0=E6=8D=AE 
 =
; } 
 ]} 
 5.19 =E7=A6=8F=E5=BD=A9=E6=8E=92=E5=88=973D=E5=
=BC=80=E5=A5=96=E6=95=B0=E6=8D=AE =E6=8E=A5=E5=8F=A3=EF=BC=9A api/fc_pl3d_lottery_logs/?limit=3D30&pag=
e=3D1 
 =E8=BF=94=E5=9B=9E=E7=A4=BA=E4=BE=8B=EF=BC=9A 
 { 
 =
; "data": [ 
 { 
 =
 "id": 11, 
 &=
nbsp; "created": "2025-04-09T13:10=
:00+00:00", 
 =
;"block_time": "2025-04-09T13:10:00+00:00", 
 &nb=
sp; "block_num": "102025089", # =E5=8C=BA=E5=
=9D=97=E9=AB=98=E5=BA=A6 
 &nb=
sp; "block_hash": "6,5,8,20883", # =E5=8C=BA=E5=9D=97=E5=
=93=88=E5=B8=8C 
 &=
nbsp;"periods": "102025089", # =E6=9C=9F=E5=8F=B7 
 &=
nbsp; "extra": { 
 &=
nbsp; "nex=
t_period_time": "2025-04-10T13:10:00+00:00" # =E4=B8=8B=E6=9C=9F=E5=BC=80=
=E5=A7=8B=E6=97=B6=E9=97=B4 
 &=
nbsp; }, &nb=
sp; "last_fc_pl3d_num": "4,3,1" # =E4=B8=8B=E6=9C=9F=E6=8E=92=E5=88=97=
3D=E5=BC=80=E5=A5=96=E6=95=B0=E6=8D=AE 
 &n=
bsp;} 
 &n=
bsp; ]} 
 6 =E5=85=AC=E5=91=8A=E6=8E=A5=E5=8F=A3 =
 =E6=9A=82=E6=97=A0 
 =
 7 =E5=B9=B3=E5=8F=B0=E5=
=BD=A9=E7=A7=8D=E5=BC=80=E5=A5=96=E7=9B=B8=E5=85=B3=E4=BF=A1=E6=81=AF =E5=BC=80=E5=A5=96=E6=95=B0=E6=8D=AE=
=E4=BB=8E websocket=E6=8E=A8=E9=80=81 
 7.1 =E6=B3=A2=E5=9C=BA=E5=8C=BA=
=E5=9D=97=E6=95=B0=E6=8D=AE =E6=
=AF=8F=E4=B8=AA=E5=8C=BA=E5=9D=97=E6=8E=A8=E9=80=81 
 {"message": 
 {"type":"block", # =
=E6=B3=A2=E5=9C=BA=E5=8C=BA=E5=9D=97=E7=B1=BB=E5=9E=8B=EF=BC=8C=E6=AF=8F=E4=
=B8=AA=E5=8C=BA=E5=9D=97=E6=8E=A8=E9=80=81 
 "block_num":71205691, 
 "block_hash":"00000000043=
e833b99a389f66da6beaea6b3a8a41ff7036b49f712d20c51a7e5", 
 "xy_ds":"
  u5355", 
 "ws_ds":"
  u5355=
", 
 "last=
_num":"5", 
 "wz_nn":"
  u95f2", "nn":"
  u95f2", 
 "xy_zx":"
  u5e84", 
 "xy_hx":"
  u8d62", 
 "ws_dx":"
  u5927", 
 "last11_5_num":"02,01,10,=
07,05", 
 "last_pk10_num":"10,02,09,06,04,03,01,08,05,07", 
 "last_k3_num":"3,1,5", 
 "created":"2025-0=
4-10T07:41:42+00:00"} =

 } 
 7.2 =E6=B3=A2=E5=9C=BA=E5=BD=A9=E7=A5=A8=E5=8C=BA=E5=9D=97=
=E6=95=B0=E6=8D=AE { 
 "send": true, 
 "message": =
{ 
 =
 "type": "lottery_v2_broadcast", 
 "block_nu=
m": 71205698, 
 "block_hash": "00000000043e8342b1dc5319fb3713=
7a9f91ff98a9dd92afdbe874a4adb88d1e", 
 =
 "created": "2025-04-10=
T07:42:03+00:00", 
=
 "last5_num": "44881", # =E6=9E=81=
=E9=80=9F 1=EF=BC=8C3=EF=BC=8C5=E5=88=86=E5=BD=A9=E5=BC=80=E5=A5=96=E6=95=
=B0=E6=8D=AE 
 "last11_5_num": "04,10,11,08,01", # 11=E9=80=
=895=E5=BC=80=E5=A5=96=E6=95=B0=E6=8D=AE 
 "last_pk10_num": "=
09,01,03,05,06,04,08,07,10,02", # pk10=E5=BC=80=E5=A5=96=E6=95=B0=E6=
=8D=AE 
 &=
nbsp; "last_k3_num": "2,4,5", 
 "lottery_log=
033": { # =E6=B3=A2=E5=9C=BA=E6=9E=81=E9=80=9F=E8=B5=9B=E8=BD=A6=EF=BC=8C=
=E6=B3=A2=E5=9C=BA=E6=9E=81=E9=80=9F=E5=BF=AB=E4=B8=89 =E6=B3=A2=E5=9C=BA=
=E6=9E=81=E9=80=9F=E5=BD=A9=EF=BC=88=E5=89=8D=E7=AB=AF=E5=8F=AB=E5=93=88=E5=
=B8=8C=E6=9E=81=E9=80=9F=E5=BD=A9=EF=BC=89=E7=AD=89 
 &=
nbsp;"periods": "105202504101884", # =E5=BD=93=E5=89=8D=E6=9C=9F=E5=8F=B7 
 &nb=
sp; "next_periods": "105202504101885" # =E4=B8=
=8B=E4=B8=80=E6=9C=9F=E5=8F=B7 
 }, 
 "lottery_=
log103": { # =E6=B3=A2=E5=9C=BA11=E9=80=895=EF=BC=8C=E6=B3=A2=E5=9C=BA1=E5=
=88=86=E5=BF=AB=E4=B8=89 =E6=B3=A2=E5=9C=BA1=E5=88=86=E5=BD=A9=EF=BC=88=E5=
=89=8D=E7=AB=AF=E5=8F=AB=E6=B3=A2=E5=9C=BA1=E5=88=86=E5=BD=A9=E6=9E=81=E9=
=80=9F=EF=BC=89=E7=AD=89 
 "periods": "111202504=
100942", 
 "next_periods": "111202504100943" 
 =
; }, 
 "lottery_log303": {=E3=80=82# =E6=B3=A2=E5=
=9C=BA3=E5=88=8611=E9=80=895 =E6=B3=A2=E5=9C=BA3=E5=88=86=E5=BF=AB=E4=B8=89=
 =E6=B3=A2=E5=9C=BA3=E5=88=86=E5=BD=A9 =EF=BC=88=E5=89=8D=E7=AB=AF=E5=8F=AB=
=E6=B3=A2=E5=9C=BA3=E5=88=86=E5=BD=A9=E6=9E=81=E9=80=9F=EF=BC=89=E7=AD=89 
 &nb=
sp; "periods": "113202504100314", 
 &=
nbsp; "next_periods": "113202504100315" 
 } 
 &n=
bsp; =E4=BA=94=E5=88=86=E7=9A=84=E6=95=B0=E6=8D=AE=E5=
=A6=82=E6=9E=9C=E6=9C=89=E5=B0=B1=E6=98=AF lottery_log503:{=E6=9C=9F=
=E5=8F=B7=E6=95=B0=E6=8D=AE=E5=90=8C=E4=B8=8A} 
 =E5=93=88=E5=B8=8C=E6=9E=81=E9=80=9F=E5=BD=A9=EF=BC=88=E5=89=
=8D=E7=AB=AF=E5=8F=AB=E6=B3=A2=E5=9C=BA=E6=9E=81=E9=80=9F=E5=BD=A9=EF=BC=89=
 lottery_log05:{=E6=9C=9F=E5=8F=B7=E6=95=B0=E6=8D=AE=E5=90=8C=E4=B8=
=8A} 
 &n=
bsp; =E8=BF=99=E4=BA=9B=E6=95=B0=E6=
=8D=AE=E9=83=BD=E6=98=AF=E8=BF=99=E4=B8=AA=E6=97=B6=E9=97=B4=E6=9C=89=E6=89=
=8D=E6=9C=89 } 
 } 7.3 =E5=89=8D=E7=AB=AF=E5=90=8D=E5=AD=97=E6=B3=A2=E5=9C=
=BA1=E5=88=86=E5=BD=A9 =E6=B3=A2=E5=9C=BA3=E5=88=86=E5=BD=A9 =E6=B3=A2=E5=
=9C=BA5=E5=88=86=E5=BD=A9 =E5=AF=B9=E5=BA=94=E7=9A=84websocket {"message":{ =

 "type":"lottery1_wsds", # =E6=B3=A2=E5=9C=BA1=E5=88=86=
=E5=BD=A9 1=E5=88=86=E5=BD=A9=E5=B0=BE=E6=95=B0=E5=8D=95=E5=8F=8C=E7=AD=89 =
=E5=85=AC=E7=94=A8=E6=AD=A4=E7=B1=BB=E5=9E=8B 
 "block_num":71205960, 
 "block_hash":"000000000=
43e8448502bda4449ad5a39d694604cfaf944e967d7643c2eb76dc6", 
 "ws_ds":"
  u53cc", #=
 =E5=B0=BE=E6=95=B0=E5=8D=95=E5=8F=8C 
 "last_num":"6", # =E6=9C=80=E5=90=8E=E4=B8=80=
=E4=BD=8D 
 "last5_num":"32766", # =E6=9C=80=E5=90=8E5=E4=BD=8D=E6=95=B0=E5=AD=
=97 
 "cre=
ated":"2025-04-10T07:55:09+00:00", =
 
 "periods":"1011867600298", # =E5=BD=93=E5=89=8D=
=E6=9C=9F=E5=8F=B7 "next_periods":"1011867600299" # =E4=B8=8B=E4=B8=80=E6=9C=9F=E5=
=8F=B7 
 }=
} 
 {"mess=
age":{ 
 "=
type":"lottery3_wsds", # =E6=B3=A2=E5=9C=BA3=E5=88=86=E5=BD=A9 3=E5=
=88=86=E5=BD=A9=E5=B0=BE=E6=95=B0=E5=8D=95=E5=8F=8C=E7=AD=89 =E5=85=AC=E7=
=94=A8=E6=AD=A4=E7=B1=BB=E5=9E=8B 
 "block_num":71205960, 
 "block_hash":"00000000043e8448502bd=
a4449ad5a39d694604cfaf944e967d7643c2eb76dc6", 
 "ws_ds":"
  u53cc", 
 "last_num":"6", 
 "last5_num":"3276=
6", 
 "cre=
ated":"2025-04-10T07:55:09+00:00", =
 
 "periods":"3011867600099", 
 "next_periods":"3011867600100=
" 
 }} 
 
 {"message":=
{ 
 "type"=
:"lottery5_wsds", # =E6=B3=A2=E5=9C=BA5=E5=88=86=E5=BD=A9 5=E5=88=86=E5=BD=
=A9=E5=B0=BE=E6=95=B0=E5=8D=95=E5=8F=8C=E7=AD=89 =E5=85=AC=E7=94=A8=E6=AD=
=A4=E7=B1=BB=E5=9E=8B =

 "block_num":71206000, 
 "block_hash":"00000000043e8470f0ffe5c4206b33327=
dc73ad7d3e06f29c352470678d649f6", 
 "ws_ds":"
  u53cc", 
 "last_num":"6", 
 "last5_num":"86496", 
 "created":"2025-04=
-10T07:57:09+00:00", "periods":"5011867600060", =
 
 "next_periods":"5011867600061" 
 }} 
 
 7.4 =E4=BB=A5=E5=
=A4=AA=E5=9D=8A=E5=8C=BA=E5=9D=97=E6=95=B0=E6=8D=AE {"=
send":true, 
 "message":{ # =E5=BD=93=E5=89=8D=E6=9C=9F=E7=9A=84=E5=BC=80=E5=A5=96=E6=
=95=B0=E6=8D=AE 
 "type":"eth_lottery_v2_broadcast", 
 "block_num":22237176, 
 "block_hash":"0x7a51fcce1=
49e72663c753f6ae646394f9c9107f4048a1f67c5020a26652cfcdc", 
 "created":"2025-04-10T07:=
57:11+00:00", 
 "last5_num":"26652", # 1=EF=BC=8C3=EF=BC=8C5=EF=BC=8C=E6=9E=81=
=E9=80=9F=E6=95=B0=E6=8D=AE 
 "last11_5_num":"07,10,06,05,02", # 11x5=E6=95=B0=
=E6=8D=AE 
 "last_pk10_num":"03,02,01,07,09,06,05,04,10,08", # pk10=E6=95=B0=E6=8D=AE=
 
 "last_k=
3_num":"3,4,3", # =E5=BF=AB=E4=B8=89=E6=95=B0=E6=8D=AE 
 "last10_6_num":"75020"=
, # =E6=96=B0=E4=BB=A5=E5=A4=AA=E5=9D=8A=E6=95=B0=E6=8D=AE 
 "eth_lottery_log":=
 # =E4=BB=A5=E5=A4=AA=E5=9D=8A=E6=9E=81=E9=80=9F=E5=BD=A9 
 { 
 "periods":"22237176", # =E5=BD=
=93=E5=89=8D=E6=9C=9F=E5=8F=B7 
 "next_periods":"22237177" # =E4=B8=8B=E4=B8=80=E6=
=9C=9F=E5=8F=B7 
 }, 
 "eth_lottery_log01": # =E4=BB=A5=E5=A4=AA=E5=9D=8A=E4=B8=80=E5=88=
=86=E5=BD=A9 =E6=96=B0=E4=BB=A5=E5=A4=AA=E5=9D=8A=E4=B8=80=E5=88=86=E5=BD=
=A9 
 { 
 "periods":=
"11111202504100957", "next_periods":"11111202504100958" 
 }, 
 "eth_lottery_log03": # =E4=BB=A5=
=E5=A4=AA=E5=9D=8A=E4=B8=89=E5=88=86=E5=BD=A9 
 { =
 
 "periods":"11113202504100319", 
 "next_periods":"1111=
3202504100320" 
 } 
 =E4=BA=94=E5=88=86=E5=BD=A9=E6=95=B0=E6=8D=AE eth_lottery_log05=EF=BC=9A{=E6=9C=9F=E5=8F=B7=E5=90=8C=E4=B8=8A} 
 }} 
 7.5 =E5=B8=
=81=E5=AE=89=E5=8C=BA=E5=9D=97=E6=95=B0=E6=8D=AE { 
 &nb=
sp;"type": "bsc_lottery_v2_broadcast", 
 "block_num": 48230001, 
 "bloc=
k_hash": "0x89823b825b8fed80d23f946bd7fe23782f86333c548f4ce1706bcd47902dcc0=
6", 
 &nbs=
p; "created": "2025-04-10T07:57:05+00:00", 
 "last5_num": "90206", 
 &nb=
sp;"last11_5_num": "04,07,09,02,06", 
 =
 "last_pk10_num": "04,03,09,06,07,0=
1,08,05,10,02", 
 "last_k3_num": "4,3,3", 
 "last10_6_num": "70647", 
 &nbs=
p;"bsc_lottery_log01": { # =E5=B8=81=E5=AE=891=E5=88=86 
 "=
periods": "10111202504100957", 
 "next_periods": "10111202504=
100958" 
 }, 
 "bsc_lottery_log03": { # =E5=B8=81=E5=AE=893=E5=88=86 
 &n=
bsp; "periods": "10113202504100319", 
 "next_perio=
ds": "10113202504100320" 
 } 
 =E5=B8=81=E5=AE=
=89=E4=BA=94=E5=88=86 bsc_lottery_log05:{=E6=9C=9F=E5=8F=B7=E5=A6=82=
=E4=B8=8A} 
 } 
 7.6 =E5=8F=B0=E6=B9=BE=E5=BD=A9=E7=A5=A8=E6=95=B0=E6=8D=AE =
 { 
 "send": true, 
 "message": { 
 "type": "tw_=
lottery_v2_broadcast", 
 "block_num": 114020204, 
 &n=
bsp;"block_hash": "15,76,38,56,55,14,05,33,35,44,72,68,11,43,58,65,19,12,29=
,17", 
 &n=
bsp; "created": "2025-04-10T07:55:00+00:00", 
 &nb=
sp;"last_tw5_num": "61651", 
 "last_tw_pk10_num": "02,05,04,0=
1,06,09,10,03,08,07", =

 "last_tw28_num": "3,9,2", 
 &=
nbsp;"tw_lottery_log": { # =E5=8F=B0=E6=B9=BE=E5=BD=A9=E7=A5=A8 
 &=
nbsp; "periods": "114020204", # =E5=BD=93=E5=89=8D=E6=9C=9F=E5=
=8F=B7 
 &=
nbsp; "next_periods": "114020205", # =E4=
=B8=8B=E4=B8=80=E6=9C=9F=E5=8F=B7 
 "next_period_=
time": "2025-04-10T08:00:00+00:00" # =E4=B8=8B=E4=B8=80=E6=9C=9F=E5=BC=80=
=E5=A7=8B=E6=97=B6=E9=97=B4 
 } 
 } 
 } 
 =
 7.7 =E7=A6=8F=E5=BD=A93D=E6=
=95=B0=E6=8D=AE =
 { 
 "send": true, =
 
 &n=
bsp; "message": { 
 "type": "fc3=
d_lottery_v2_broadcast", 
 "block_nu=
m": "42438", 
 "block_hash": "4,3,2",=
 
 &=
nbsp; "created": "2025-03-29T03:18:00+00=
:00", 
 &n=
bsp; "last_fc3d_num": "4,3,2", 
 =
; "fc3d_lottery_log": { 
 =
 "periods": "42438", 
 &nb=
sp; "next_periods": "4=
2439", 
 &=
nbsp; "nex=
t_period_time": "2025-03-29T03:19:00+00:00" 
 &nb=
sp; } 
 } =
 
 7.8 =E6=8E=92=E5=88=9735=E6=95=B0=E6=8D=AE { "send": true, 
 "message"=
: { 
 &nbs=
p; "type": "pl35_lottery_v2_broadc=
ast", 
 &n=
bsp; "block_num": "568106", =
 
 &n=
bsp; "block_hash": "51317", 
 &=
nbsp; "created": "2025-03-29T04:26:00+00:00", 
 &n=
bsp; "last_pl35_num": "51317", 
 =
; "pl35_lottery_log": { 
 =
 "periods": "568106", 
 &n=
bsp; "next_periods": "568107", 
 =
; "next_period_time": =
"2025-03-29T04:27:00+00:00" 
 } 
 =
; } 
 } 
 7.9 =E7=A6=8F=E5=BD=A9=E6=8E=92=E5=88=973D=E6=95=B0=E6=8D=AE { "send": true, 
 "message"=
: { 
 &nbs=
p; "type": "fc_pl3d_lottery_v2_bro=
adcast", 
 "block_num": "42506", 
 =
; "block_hash": "5,1,3,51317", 
 =
; "created": "2025-03-29T04:26:00+00:00", 
 =
; "last_fc_pl3d_num": "8,2,0", 
 =
; "fc_pl3d_lottery_log": { 
 &=
nbsp; "periods": "42506", 
 &nbs=
p; "next_periods": "42=
507", 
 &n=
bsp; "next=
_period_time": "2025-03-29T04:27:00+00:00" 
 &nbs=
p; } 
 } 
 } =

 8. =E5=B9=B3=E5=8F=B0=E5=BD=A9=E7=A7=8D=E6=95=B0=E6=
=8D=AE=E5=92=8C=E5=90=8D=E7=A7=B0ID =E5=8F=82=E8=80=83 4=E9=87=8C=E4=BB=A5=E6=8F=90=E4=BE=9B 
 9. =E5=
=BD=A9=E7=A5=A8=E6=8A=95=E6=B3=A8=E8=AE=B0=E5=BD=95=E6=8E=A5=E5=8F=A3 =E6=8E=A5=E5=8F=A3=EF=BC=9A /ap=
i/web_bets/ GET 
 =E5=8F=82=E6=95=B0=EF=BC=9A 
 =E9=9C=80=E8=A6=81=E6=B3=A8=E6=84=8F=E7=9A=84=E6=98=AF=EF=BC=8C=E6=9C=89=
=E5=87=A0=E4=B8=AA=E5=90=8D=E5=AD=97=E6=88=91=E6=96=B9=E5=89=8D=E7=AB=AF=E8=
=BF=9B=E8=A1=8C=E4=BA=86=E8=87=AA=E5=AE=9A=E4=B9=89 
 =E6=
=B3=A2=E5=9C=BA=E6=9E=81=E9=80=9F=E5=BD=A9=EF=BC=9A =E9=9C=80=E8=A6=81=E4=BC=A0=E9=80=92 =
=E5=93=88=E5=B8=8C=E6=9E=81=E9=80=9F=E5=BD=A9 
 =E6=B3=A2=
=E5=9C=BA 1=E5=88=86=E5=BD=A9=EF=BC=9A =E9=9C=80=E8=A6=81=E4=BC=A0=E9=
=80=92 =E5=93=88=E5=B8=8C=E4=B8=80=E5=88=86=E5=BD=A9 
 =E6=B3=A2=
=E5=9C=BA 3=E5=88=86=E5=BD=A9=EF=BC=9A =E9=9C=80=E8=A6=81=E4=BC=A0=E9=
=80=92 =E5=93=88=E5=B8=8C=E4=B8=89=E5=88=86=E5=BD=A9 
 =E6=B3=A2=
=E5=9C=BA 5=E5=88=86=E5=BD=A9=EF=BC=9A =E9=9C=80=E8=A6=81=E4=BC=A0=E9=
=80=92 =E5=93=88=E5=B8=8C=E4=BA=94=E5=88=86=E5=BD=A9 
 =E5=93=88=
=E5=B8=8C=E6=9E=81=E9=80=9F=E5=BD=A9=EF=BC=9A=E9=9C=80=E8=A6=81=E4=BC=A0=E9=
=80=92 =E6=B3=A2=E5=9C=BA=E6=9E=81=E9=80=9F=
=E5=BD=A9 
 =E5=93=88=E5=B8=8C 1=E5=88=86=E5=BD=A9=
=E6=9E=81=E9=80=9F=EF=BC=9A=E9=9C=80=E8=A6=81=E4=BC=A0=E9=80=92 =E6=B3=A2=
=E5=9C=BA=E4=B8=80=E5=88=86=E5=BD=A9 
 =E5=93=88=E5=B8=8C =
3=E5=88=86=E5=BD=A9=E6=9E=81=E9=80=9F=EF=BC=9A=E9=9C=80=E8=A6=81=E4=BC=A0=
=E9=80=92 =E6=B3=A2=E5=9C=BA=E4=B8=89=E5=88=86=E5=BD=A9 
 =E5=
=93=88=E5=B8=8C 5=E5=88=86=E5=BD=A9=E6=9E=81=E9=80=9F=EF=BC=9A=E9=9C=
=80=E8=A6=81=E4=BC=A0=E9=80=92 =E6=B3=A2=E5=9C=BA=E4=BA=94=E5=88=86=E5=BD=
=A9 
 =E5=85=B6=E4=BB=96=E5=90=8D=E7=A7=B0=E7=9B=B8=E5=90=8C 
 1. filters: =
 
 2. 
 { 
 "game_name":"=E5=93=88=E5=B8=8C=E4=B8=80=E5=88=86=E5=BD=A9", &nb=
sp;# =E5=BD=A9=E7=A7=8D=E5=90=8D=E7=A7=B0 
 "bet_time_dr":"2025=
-04-10~2025-04-11" # =E6=8A=95=E6=B3=A8=E7=9A=84=E6=97=B6=E9=97=B4=E8=8C=83=
=E5=9B=B4 
 } 
 3. 
 4. 
 limit=
: 
 5. 
 20 
 6. 
 7. 
 page: 
 8. =

 1 
 9. 
 10.=E5=BD=A9=E7=A5=A8=
=E6=92=A4=E5=8D=95=E6=8E=A5=E5=8F=A3 =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/web_bets/lott/cancel/<=E8=
=A6=81=E6=92=A4=E9=94=80=E7=9A=84=E6=B3=A8=E5=8D=95ID> POST 
 11.=E5=BD=
=A9=E7=A7=8D=E6=8A=95=E6=B3=A8=E6=8E=A5=E5=8F=A3 =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/web_bets/lott POS=
T 
 =E6=89=80=E6=9C=89=E5=BD=A9=E7=A5=A8=E6=8A=95=E6=B3=A8=E5=9D=
=87=E4=B8=BA=E6=AD=A4=E6=8E=A5=E5=8F=A3=EF=BC=8C=E6=B8=B8=E6=88=8F id=
=E4=B8=8D=E5=90=8C=EF=BC=8C=E8=A7=84=E5=88=99ID=E4=B8=8D=E5=90=8C=EF=BC=8C =
=E5=85=B7=E4=BD=93=E6=8A=95=E6=B3=A8=E5=86=85=E5=AE=B9=E5=8F=AF=E4=BB=8E=E6=
=B5=8B=E8=AF=95=E7=8E=AF=E5=A2=83=E5=8F=82=E8=80=83 https://hash.iyes.dev 
 { 
 "au=
to_type": "=E6=8A=95=E6=B3=A8=E6=9D=A5=E6=BA=90=EF=BC=8C10=E4=B8=AA=E9=95=
=BF=E5=BA=A6=E4=B9=8B=E5=86=85=E7=9A=84=E5=AD=97=E7=AC=A6=E4=B8=B2", =
 
 "b=
et_contents": [ # =E6=AF=8F=E4=B8=AAdict=E4=B8=80=E4=B8=AA=E6=B3=A8=E5=8D=
=95=E6=95=B0=E6=8D=AE =

 { =
 
 "rule_i=
d": "13", # =E8=A7=84=E5=88=99ID 
 "bet_con=
tent": ",,,13579,", # =E6=8A=95=E6=B3=A8=E5=86=85=E5=AE=B9 
 =
 "amount_unit": 2, # =E5=8D=95=E5=85=83=E9=87=91=E9=
=A2=9D 
 &=
nbsp; "bets_nums": 5, # =E6=8A=95=E6=B3=A8=E6=
=B3=A8=E6=95=B0 
 "multiple": 2, # =E6=8A=
=95=E6=B3=A8=E7=9A=84=E5=80=8D=E6=95=B0 =
 
 "bet_am=
ount": 20, # =E6=8A=95=E6=B3=A8=E7=9A=84=E9=87=91=E9=A2=9D 
 =
 "solo": false, # =E6=98=AF=E5=90=A6=E5=8D=95=E6=8C=
=91 
 &nbs=
p; "min_single_bet_bonus": 38.8 # =E6=AF=
=8F=E6=B3=A8=E4=B8=AD=E5=A5=96=EF=BC=88=E9=9D=9E=E5=BF=85=E8=A6=81=E5=8F=82=
=E6=95=B0=EF=BC=89 }, 
 { 
 &=
nbsp; "rule_id": "13", 
 "bet_content": ",02=
468,,,", 
 "amount_unit": 2, 
 &=
nbsp; "bets_nums": 5, 
 "multiple": 2, 
 =
; "bet_amount": 20, 
 "sol=
o": false, 
 "min_single_bet_bonus": 38.8 
 &nb=
sp; }, 
 { 
 "rule_id": "13", 
 &n=
bsp; "bet_content": "13579,,,,", 
 &=
nbsp; "amount_unit": 2, =
 
 "bets_nums": 5, 
 &nbs=
p; "multiple": 2, 
 "bet_a=
mount": 20, 
 "solo": false, 
 &nbs=
p; "min_single_bet_bonus": 38.8 
 =
 } 
 ], 
 "game_id"=
: 29, # =E6=B8=B8=E6=88=8FID =
 
 "currency": 3, # =E5=B8=81=E7=
=A7=8D 0 usdt 1 trx 3 cny 
 "bet_multiple": [ # =E5=A4=96=E5=B1=82=E4=B8=
=8D=E5=8A=A0=E5=80=8D=E6=97=B6=EF=BC=8C=E5=8F=AF=E4=BB=A5=E4=BC=A0=E9=80=92=
=E4=B8=BA[] 
 { 
 "bet_amount": 60,=
 # =E6=80=BB=E9=87=91=E9=A2=9D 
 =
 "multiple"=
: 1 # =E5=A4=96=E5=B1=82=E5=8A=A0=E5=80=8D=E5=80=8D=E6=95=B0 
 &nbs=
p;} 
 &nbs=
p; ] 
 } 
 12.=E6=B5=8B=E8=AF=95=E8=B4=A6=E6=88=B7 =E8=B4=A6=E6=88=B7 /=E5=AF=86=E7=A0=81 
 testcq01 
 testcq01 
 testcq02 
 testcq02=
 
 testcq03 
 =
testcq03 
 =
testcq04 
 testcq04 
=
 testcq05 
 testcq05 
 testcq06 
 testcq06 
 testcq07 
 testcq07 
 testcq08 
 testcq08 
 testcq09 
 testcq09 
 testcq10 
 testcq10 
 =E8=B5=84=E9=87=91=E5=AF=86=E7=A0=81=EF=BC=9A 147258 
 =E5=AF=86=E4=BF=9D=E7=AD=94=E6=A1=88=EF=BC=9A 147258 
 13=
. =E5=B9=B3=E5=8F=B0=E5=BD=93=E6=9C=9F=E6=B3=A8=E5=8D=95=E9=9C=80=E9=97=B4=
=E9=9A=94=E5=87=A0=E7=A7=92=E6=89=8D=E5=8F=AF=E4=BB=A5=E6=8A=95=E6=B3=A8=E4=
=BB=A5=E5=8F=8A=E5=85=B6=E4=BB=96=E6=B3=A8=E6=84=8F=E4=BA=8B=E9=A1=B9 =E5=BD=93=E6=9C=9F=E6=9C=80=E5=90=8E 3=E7=A7=92=E6=97=A0=E6=B3=95=E6=8A=95=E6=B3=A8 
 14. =E5=8D=95 ip=E8=83=BD=E4=B8=8D=E8=
=83=BD=E7=99=BB=E5=BD=95=E5=A4=9A=E8=B4=A6=E5=8F=B7=E3=80=81=E6=98=AF=E5=90=
=A6=E6=9C=89=E8=B4=A6=E5=8F=B7=E4=B8=8A=E9=99=90=E9=99=90=E5=88=B6=E4=BB=A5=
=E5=8F=8A=E6=9C=89=E6=B2=A1=E6=9C=89=E5=85=B6=E4=BB=96=E9=99=90=E5=88=B6=E4=
=BA=8B=E9=A1=B9 =E5=8D=95 =
ip=E6=97=A0=E8=B4=A6=E6=88=B7=E9=99=90=E5=88=B6 
 15.=E8=B4=A6=E5=8F=B7=E7=99=BB=E5=BD=95=E8=BF=9E=E7=BB=AD=
=E8=BE=93=E9=94=99=E5=AF=86=E7=A0=81=E6=9C=BA=E5=88=B6 =
 =E8=BF=9E=E7=BB=AD=E9=94=99=E8=AF=AF 5=E6=AC=
=A1=E5=B0=86=E9=94=81=E5=AE=9A2=E5=B0=8F=E6=97=B6 
 16. =E5=8D=95=E4=B8=AA ip=E6=9C=89=E6=B2=
=A1=E6=9C=89cdn=E8=AF=B7=E6=B1=82=E6=95=B0=E9=99=90=E5=88=B6=E3=80=81=E4=B8=
=80=E7=A7=92=E9=99=90=E5=88=B6=E5=A4=9A=E5=B0=91=E6=AC=A1=E3=80=81=E5=A6=82=
=E6=9C=89=E9=99=90=E5=88=B6=E8=83=BD=E4=B8=8D=E8=83=BD=E5=8A=A0ip=E7=99=BD=
=E5=90=8D=E5=8D=95 =E4=B8=8B=E6=
=B3=A8=E6=8E=A5=E5=8F=A3=E6=97=A0=E9=99=90=E5=88=B6 
 17.token =E5=8F=
=AF=E4=BB=A5=E4=BF=9D=E5=AD=98=E5=A4=9A=E4=B9=85 =E4=B8=80=E5=B9=B4 
 18.=E8=83=BD=E4=B8=8D=E8=83=BD=E4=B8=80=E6=9D=A1=E4=B8=8D=E5=
=AF=B9=E5=A4=96=E5=BC=80=E6=94=BE=E7=9A=84=E4=BA=91=E7=AB=AF=E6=8C=82=E6=9C=
=BA=E4=B8=93=E7=94=A8=E7=BA=BF=E8=B7=AF =E5=BC=80=E5=8F=91=E5=AE=8C=E6=88=90=E5=90=8E=EF=BC=8C=E8=B7=9F=E4=
=B8=BB=E7=AE=A1=E8=B0=88=EF=BC=8C=E4=B8=BB=E7=AE=A1=E5=90=8C=E6=84=8F=E5=90=
=8E=E4=BC=9A=E5=AE=89=E6=8E=92=E8=BF=90=E7=BB=B4=E4=BA=BA=E5=91=98=E9=85=8D=
=E7=BD=AE 
 19. LOGO=E7=AD=89 =E4=B8=BB=E8=89=B2=EF=BC=9A 5180F6 
 =E5=BA=
=95=E5=B1=82 BG=EF=BC=9AF6F9FE 
 =E6=96=87=E5=AD=97=E4=B8=
=BB=E8=89=B2=EF=BC=9A 575E71 
 =E6=8F=8F=E8=BF=
=B0=E6=96=87=E5=AD=97=EF=BC=9A 909FBB 
 20. =E6=9C=AA=E6=9D=A5=E5=BC=
=80=E7=9B=98=E4=BF=A1=E6=81=AF =
=E6=AD=A4=E6=95=B0=E6=8D=AE=E4=BB=85=E4=BE=9B=E5=8F=82=E6=95=B0=EF=BC=8C=E4=
=B8=8D=E4=BD=9C=E4=B8=BA=E5=AE=9E=E9=99=85=E5=BC=80=E5=A5=96=E6=97=B6=E9=97=
=B4 
 =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/web_bets/lot=
t/periods 
 =
 =E5=8F=82=E6=95=B0=EF=BC=9A 
 game_id =E6=B8=B8=E6=88=8FI=
D 
 num_peri=
ods =E9=9C=80=E8=A6=81=E6=9C=AA=E6=9D=A5=E5=87=A0=E6=9C=9F 
 =E8=
=BF=94=E5=9B=9E=E6=95=B0=E6=8D=AE=EF=BC=9A 
 {"code":201,"data":[ 
 { 
 "period":"1011867900379",=
 
 "start_=
time":"2025-04-10 09:15:13", =
 
 "end_time":"2025-04-10 09:16:13" 
 }, 
 { 
 "period":"1011867900380", 
 "start_time":"2025-04-=
10 09:16:13", 
 "end_time":"2025-04-10 09:17:13" 
 }, 
 { 
 "period":"1011867900381", 
 "start_time":"2025-04-10 09:17:13", 
 "end_time=
":"2025-04-10 09:18:13" 
 }, { 
 "period":"1011867900382", 
 "start_time":"2025-04-10 09:18:13", 
 "end_time":"2025-04-10=
 09:19:13" 
 }, 
 {"=
period":"1011867900383", 
 "start_time":"2025-04-10 09:19:13", 
 "end_time":"2025-04-10 09:20:=
13" 
 } 
 ],"message=
":"success"} 
 =E5=85=A8=E5=B1=80=E9=94=99=E8=AF=AF=
=E7=A0=81=EF=BC=9A 400 =E5=8F=82=E6=95=B0=E6=95=
=B0=E6=8D=AE 
 401 =E9=9C=80=E8=A6=81=E7=99=BB=E9=99=86=E6=89=8D=E5=8F=AF=E4=BB=A5=E8=
=AE=BF=E9=97=AE 
 429 =E9=A2=91=E7=8E=87=E8=AF=B7=E6=B1=82=E8=B6=85=E8=BF=87=E9=99=90=
=E5=88=B6 
 =
500+ =E6=9C=8D=E5=8A=A1=E5=99=A8=E9=94=99=E8=AF=AF 
 
------=_NextPart_000_0076_01C29953.BE473C30
Content-Type: application/octet-stream;
Content-Transfer-Encoding: base64
Content-Location: file:///Users/a01/Library/Containers/com.kingsoft.wpsoffice.mac/Data/tmp/wps-a01/~tmp{da1c6af2-c5b9-4cbd-8c49-05b4a27f0b5c}5341681360.files/filelist.xml

PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0iVVRGLTgiIHN0YW5kYWxvbmU9InllcyI/Pg0K
PHhtbCB4bWxuczpvPSJ1cm46c2NoZW1hcy1taWNyb3NvZnQtY29tOm9mZmljZTpvZmZpY2UiPjxv
Ok1haW5GaWxlIEhSZWY9Ii4uL350bXB7ZGExYzZhZjItYzViOS00Y2JkLThjNDktMDViNGEyN2Yw
YjVjfTUzNDE2ODEzNjAiLz48bzpGaWxlIEhSZWY9ImZpbGVsaXN0LnhtbCIvPjwveG1sPg==

------=_NextPart_000_0076_01C29953.BE473C30--


## Filtered lines

· 17.token =E5=8F=AF=E4=BB=A5=E4=BF=9D=E5=AD=98=E5=A4=9A=E4=B9=85
=A0=E5=AF=86 : =E8=AF=B7=E6=B1=82=E6=8E=A5=E5=8F=A3 : /api/s=
=B9 https:// hash.iyes.dev /=EF=BC=8C =E7=9B=B8=E5=85=B3=E6=8E=A5=E5=8F=A3=
https://hash.iyes=
hash.iyes.dev / ?token=3D=E7=94=A8=E6=88=B7token
=99 token=E4=BC=A0=E9=80=92 Anonymous
=E5=90=8D + =E5=AF=86=E7=A0=81=EF=BC=9Ausername=E3=80=81password
=EF=BC=9APOST /api/users/i/no_login/send/email
=E6=89=8B=E6=9C=BA=EF=BC=9APOST /api/users/i=
=E5=8F=82=E6=95=B0=EF=BC=9Ausername/password =E6=88=
=9Atoken=E3=80=81refresh_token=E3=80=81username=E3=80=81token_type(Bearer)=
=E3=80=81is_temp_pwd_user
curl 'ht=
-H 'origin: https://hash.iyes.dev' \
-H 'referer: http=
s://hash.iyes.dev/' \ --data-raw '{"username":"testcq01","password":"=
curl 'https:=
-H 'origin: https://hash.iyes.dev' \
user@example.com","email_code":"123456","is_ai":true}'
curl 'https://hash-game-admin.iyes.dev/auth/login' \
-H 'origin: https://hash.iyes.de=
=A3=EF=BC=9APOST /auth/login/security
=E6=95=B0=EF=BC=9Ausername=E3=80=81password=E3=80=81new_password=E3=80=81wp=
_password=E3=80=81wp_password2=E3=80=81security_question=E3=80=81security_r=
eminder=E3=80=81security_code
=98=E5=88=97=E8=A1=A8=EF=BC=9AGET /auth/security/questions
curl 'https://hash-game-admin.iyes.dev/auth/login/security' \
-H 'origin: https://hash.iyes.dev' \ =
--data-raw '{"username":"testcq01","password":"testcq01","new_password":"1=
47258","wp_password":"147258","wp_password2":"147258","security_question":"=
=AF?","security_reminder":"147255","security_code":"147258"}'
curl 'https://hash-game-admin=
raw '{"username":"testcq01","password":"testcq01","login_key":"=E4=B8=8A=E4=
th/security/reset =
=E5=8F=82=E6=95=B0=EF=BC=9Ausername=E3=80=81password=
=E3=80=81security_key=E3=80=81=E5=8E=9F=E5=AF=86=E4=BF=9D=E3=80=81=E6=96=B0=
2.7 =E5=88=B7=E6=96=B0 Token /=
=BC=9APOST /auth/refresh/token=EF=BC=8C=E5=8F=82=E6=95=B0 refresh_token=E3=
=E6=88=96 POST /auth/logout=EF=BC=8CHeader: Authorization: Bearer {token}
=99=BB=E5=BD=95=E6=88=96token=E5=A4=B1=E6=95=88
uthorization: Bearer {token}
=E5=92=8C=E4=BD=99=E9=A2=9D=E6=8E=A5=E5=8F=A3=EF=BC=9A =E6=8E=A5=E5=8F=A3=EF=BC=9AGET /api/users/i/info =
ader=EF=BC=9AAuthorization: Bearer {token}
"username": "no8899",
T /api/agents/i/real/rate
=B5=84=E8=AE=AF=E6=8E=A5=E5=8F=A3=EF=BC=9A =E6=8E=A5=E5=8F=A3=EF=BC=9AGET /api/users/i/security_question
=E8=BF=94=E5=9B=9E=EF=BC=9Asecurity_question=E3=80=81security_reminder
=A5=E5=8F=A3=EF=BC=9AGET /api/web_bets/total
filters=EF=BC=9Abet_time_dr =E4=B8=BA=E4=BB=8A=E6=97=A5=E6=97=A5=E6=9C=9F=
"bet_time_d=
=EF=BC=9A /api/lottery_log033s GET
=E6=8E=A5=E5=8F=A3=EF=BC=9A /api/lottery_log103s?limit=3D20&page=3D1=
=A5=E5=8F=A3=EF=BC=9A /api/lottery_log303s?limit=3D20&page=3D1 =
=8F=A3=EF=BC=9A /api/lottery_log503s?limit=3D20&page=3D1
=A2=E5=9C=BA=E6=9E=81=E9=80=9F=E5=BD=A9 =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/lottery_log05s?limit=3D20&pag=
=E6=8E=A5=E5=8F=A3=EF=BC=9A /api/lottery_logs
5.7 =E6=B3=A2=E5=9C=BA=E4=B8=89=E5=88=86=E5=BD=A9 =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/lottery_log3s
=E6=B3=A2=E5=9C=BA=E4=BA=94=E5=88=86=E5=BD=A9 =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/lottery_log5s
=BB=A5=E5=A4=AA=E5=9D=8A=E6=9E=81=E9=80=9F=E5=BD=A9 =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/eth_block_logs/?l=
=EF=BC=9A /api/eth_lottery_logs/?limit=3D20&page=3D1
=EF=BC=9A /api/eth_lottery_log3s/?limit=3D20&page=3D1
=EF=BC=9A /api/eth_lottery_log5s/?limit=3D20&page=3D1
=E6=8E=A5=E5=8F=A3=EF=BC=9A /api/bsc_lottery_logs/?limit=3D20&pag=
=9A /api/bsc_lottery_log3s/?limit=3D20&page=3D1
=E6=8E=A5=E5=8F=A3=EF=BC=9A /api/bsc_lottery_log5s/?limit=3D20&pag=
5.16 =E5=8F=B0=E6=B9=BE5=E5=88=86=E5=BD=A9 =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/tw_lottery_logs/?l=
5.17 =E7=A6=8F=E5=BD=A93D=E5=BC=80=E5=A5=96=E6=95=B0=E6=8D=AE =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/fc=
3d_lottery_logs/?limit=3D10&page=3D1
=8F=A3=EF=BC=9A /api/pl35_lottery_logs/?limit=3D30&page=3D1
=BC=80=E5=A5=96=E6=95=B0=E6=8D=AE =E6=8E=A5=E5=8F=A3=EF=BC=9A api/fc_pl3d_lottery_logs/?limit=3D30&pag=
"type": "lottery_v2_broadcast",
"lottery_log=
"lottery_=
"lottery_log303": {=E3=80=82# =E6=B3=A2=E5=
=A6=82=E6=9E=9C=E6=9C=89=E5=B0=B1=E6=98=AF lottery_log503:{=E6=9C=9F=
lottery_log05:{=E6=9C=9F=E5=8F=B7=E6=95=B0=E6=8D=AE=E5=90=8C=E4=B8=
"type":"lottery1_wsds", # =E6=B3=A2=E5=9C=BA1=E5=88=86=
type":"lottery3_wsds", # =E6=B3=A2=E5=9C=BA3=E5=88=86=E5=BD=A9 3=E5=
:"lottery5_wsds", # =E6=B3=A2=E5=9C=BA5=E5=88=86=E5=BD=A9 5=E5=88=86=E5=BD=
"type":"eth_lottery_v2_broadcast",
"eth_lottery_log":=
"eth_lottery_log01": # =E4=BB=A5=E5=A4=AA=E5=9D=8A=E4=B8=80=E5=88=
"eth_lottery_log03": # =E4=BB=A5=
=E4=BA=94=E5=88=86=E5=BD=A9=E6=95=B0=E6=8D=AE eth_lottery_log05=EF=BC=9A{=E6=9C=9F=E5=8F=B7=E5=90=8C=E4=B8=8A}
sp;"type": "bsc_lottery_v2_broadcast",
p;"bsc_lottery_log01": { # =E5=B8=81=E5=AE=891=E5=88=86
"bsc_lottery_log03": { # =E5=B8=81=E5=AE=893=E5=88=86
=89=E4=BA=94=E5=88=86 bsc_lottery_log05:{=E6=9C=9F=E5=8F=B7=E5=A6=82=
lottery_v2_broadcast",
nbsp;"tw_lottery_log": { # =E5=8F=B0=E6=B9=BE=E5=BD=A9=E7=A5=A8
d_lottery_v2_broadcast",
; "fc3d_lottery_log": {
p; "type": "pl35_lottery_v2_broadc=
; "pl35_lottery_log": {
p; "type": "fc_pl3d_lottery_v2_bro=
; "fc_pl3d_lottery_log": {
i/web_bets/ GET
"bet_time_dr":"2025=
=E6=92=A4=E5=8D=95=E6=8E=A5=E5=8F=A3 =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/web_bets/lott/cancel/<=E8=
=A9=E7=A7=8D=E6=8A=95=E6=B3=A8=E6=8E=A5=E5=8F=A3 =E6=8E=A5=E5=8F=A3=EF=BC=9A /api/web_bets/lott POS=
=B5=8B=E8=AF=95=E7=8E=AF=E5=A2=83=E5=8F=82=E8=80=83 https://hash.iyes.dev
"bet_con=
nbsp; "bets_nums": 5, # =E6=8A=95=E6=B3=A8=E6=
"bet_am=
p; "min_single_bet_bonus": 38.8 # =E6=AF=
"bet_content": ",02=
nbsp; "bets_nums": 5,
; "bet_amount": 20,
"min_single_bet_bonus": 38.8
bsp; "bet_content": "13579,,,,",
"bets_nums": 5,
"bet_a=
p; "min_single_bet_bonus": 38.8
"bet_multiple": [ # =E5=A4=96=E5=B1=82=E4=B8=
"bet_amount": 60,=
17.token =E5=8F=
=E6=8E=A5=E5=8F=A3=EF=BC=9A /api/web_bets/lot=
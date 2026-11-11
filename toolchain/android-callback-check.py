#!/usr/bin/env python3
"""Check that a native browser callback returns to Minder in an Android emulator."""
import os,re,shlex,subprocess,time,xml.etree.ElementTree as ET
serial=os.environ.get('MINDER_ANDROID_SERIAL','emulator-5584')
if not serial.startswith('emulator-'):raise SystemExit('Use a disposable emulator, not a physical phone.')
prefix=shlex.split(os.environ.get('MINDER_ADB','podman exec minder-auth-emulator adb'))+['-s',serial]
def adb(*args):return subprocess.check_output(prefix+list(args),text=True,stderr=subprocess.DEVNULL)
def foreground():
 data=adb('shell','dumpsys','activity','activities')
 line=next((s for s in data.splitlines() if 'topResumedActivity=' in s),'')
 match=re.search(r'u0 ([^ ]+)',line);return match.group(1) if match else ''
def control(label):
 adb('shell','rm','-f','/sdcard/minder-callback-check.xml')
 result=adb('shell','uiautomator','dump','/sdcard/minder-callback-check.xml')
 if 'dumped' not in result:return None
 xml=adb('shell','cat','/sdcard/minder-callback-check.xml')
 for n in ET.fromstring(xml).iter('node'):
  if n.get('text')==label or n.get('content-desc')==label:
   a,b,c,d=map(int,re.findall(r'\d+',n.get('bounds')));return (a+c)//2,(b+d)//2
 return None
adb('shell','pm','clear','io.liveleds.minder')
adb('shell','am','start','-n','io.liveleds.minder/.MainActivity')
for _ in range(15):
 try:button=control('Open sign-in probe')
 except (subprocess.CalledProcessError,ET.ParseError):button=None
 if button:break
 time.sleep(1)
else:raise SystemExit('FAIL: sign-in button did not appear.')
adb('shell','input','tap',*map(str,button))
for _ in range(25):
 current=foreground()
 if current and not current.startswith('io.liveleds.minder/'):break
 time.sleep(.5)
else:raise SystemExit('FAIL: authentication did not open a browser.')
for _ in range(20):
 try:button=control('Return to Minder')
 except (subprocess.CalledProcessError,ET.ParseError):button=None
 if button:break
 time.sleep(.5)
else:raise SystemExit('FAIL: browser fixture did not load.')
adb('shell','input','tap',*map(str,button))
for _ in range(30):
 current=foreground()
 if current=='io.liveleds.minder/.MainActivity':
  try:received=control('Callback received')
  except (subprocess.CalledProcessError,ET.ParseError):received=None
  if received:
   print('PASS: real browser link returns to Minder and delivers its callback. No session is created.')
   break
 time.sleep(.2)
else:raise SystemExit('FAIL: browser callback left foreground at '+current)

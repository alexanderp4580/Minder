#!/usr/bin/env python3
"""Export native and raster Minder icons from the canonical SVG. Requires rsvg-convert."""
from pathlib import Path
import copy,subprocess,xml.etree.ElementTree as ET
root=Path(__file__).resolve().parent.parent
source=root/'brand/app-icon.svg';svg=ET.parse(source);ns={'s':'http://www.w3.org/2000/svg'}
mark=svg.find('.//s:g',ns);assert mark is not None
legacy=copy.deepcopy(svg);legacy.find('.//s:g',ns).set('transform','translate(-19.44 -19.44) scale(1.36)')
def png(target,size,maskable=False):
 target=root/target;target.parent.mkdir(parents=True,exist_ok=True)
 data=ET.tostring((svg if maskable else legacy).getroot())
 subprocess.run(['rsvg-convert','-w',str(size),'-h',str(size),'-o',str(target)],input=data,check=True)
for density,size in [('mdpi',48),('hdpi',72),('xhdpi',96),('xxhdpi',144),('xxxhdpi',192)]:png(f'app/android/app/src/main/res/mipmap-{density}/ic_launcher.png',size)
for size in [192,512]:
 png(f'app/web/icons/Icon-{size}.png',size)
 png(f'app/web/icons/Icon-maskable-{size}.png',size,True)
png('app/web/favicon.png',32);png('server/api/assets/minder-icon.png',192);png('brand/app-icon.png',512)
def vector(monochrome=False):
 body=[]
 for p in mark:
  a=p.attrib;attrs={'android:pathData':a['d'],'android:fillColor':'#00000000'}
  if a.get('fill')!='none':attrs['android:fillColor']='#ffffff' if monochrome else a['fill']
  if 'stroke' in a:attrs.update({'android:strokeColor':'#ffffff' if monochrome else a['stroke'],'android:strokeWidth':a['stroke-width'],'android:strokeLineCap':'square'})
  body.append('  <path '+' '.join(f'{k}="{v}"' for k,v in attrs.items())+'/>')
 return '<vector xmlns:android="http://schemas.android.com/apk/res/android" android:width="108dp" android:height="108dp" android:viewportWidth="108" android:viewportHeight="108">\n'+'\n'.join(body)+'\n</vector>\n'
res=root/'app/android/app/src/main/res'
(res/'drawable/ic_minder_foreground.xml').write_text(vector());(res/'drawable/ic_minder_monochrome.xml').write_text(vector(True))
(res/'values/minder_icon_colors.xml').write_text('<resources><color name="minder_icon_background">#0a0a0a</color></resources>\n')
for version in [26,33]:
 folder=res/f'mipmap-anydpi-v{version}';folder.mkdir(exist_ok=True)
 mono='\n  <monochrome android:drawable="@drawable/ic_minder_monochrome"/>' if version==33 else ''
 (folder/'ic_launcher.xml').write_text('<adaptive-icon xmlns:android="http://schemas.android.com/apk/res/android">\n  <background android:drawable="@color/minder_icon_background"/>\n  <foreground android:drawable="@drawable/ic_minder_foreground"/>'+mono+'\n</adaptive-icon>\n')
print('Exported Android adaptive, monochrome and legacy icons; web icons; favicon; public grant logo.')

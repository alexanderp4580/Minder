#!/usr/bin/env python3
"""Export native and raster Minder icons from the canonical SVG. Requires rsvg-convert."""
from pathlib import Path
import copy,subprocess,xml.etree.ElementTree as ET
root=Path(__file__).resolve().parent.parent
source=root/'brand/app-icon.svg';svg=ET.parse(source);ns={'s':'http://www.w3.org/2000/svg'}
def layer(tree,gid):return next(g for g in tree.getroot().findall('s:g',ns) if g.get('id')==gid)
def only(keep):
 tree=copy.deepcopy(svg)
 for gid in ('minder-orbit','minder-mark'):
  if gid!=keep:tree.getroot().remove(layer(tree,gid))
 return tree
legacy=copy.deepcopy(svg);layer(legacy,'minder-mark').set('transform','translate(-19.44 -19.44) scale(1.36)')
def png(target,size,maskable=False,tree=None):
 target=root/target;target.parent.mkdir(parents=True,exist_ok=True)
 data=ET.tostring((tree or (svg if maskable else legacy)).getroot())
 subprocess.run(['rsvg-convert','-w',str(size),'-h',str(size),'-o',str(target)],input=data,check=True)
for density,size in [('mdpi',48),('hdpi',72),('xhdpi',96),('xxhdpi',144),('xxxhdpi',192)]:png(f'app/android/app/src/main/res/mipmap-{density}/ic_launcher.png',size)
for size in [192,512]:
 png(f'app/web/icons/Icon-{size}.png',size)
 png(f'app/web/icons/Icon-maskable-{size}.png',size,True)
png('app/web/favicon.png',32);png('server/api/assets/minder-icon.png',192);png('brand/app-icon.png',512)
def vector():
 letter=svg.find('.//s:path[@id="letter"]',ns).get('d')
 return '<vector xmlns:android="http://schemas.android.com/apk/res/android" android:width="108dp" android:height="108dp" android:viewportWidth="108" android:viewportHeight="108">\n  <path android:pathData="'+letter+'" android:fillColor="#ffffff"/>\n</vector>\n'
res=root/'app/android/app/src/main/res'
adaptive=only('minder-orbit');next(g for g in layer(adaptive,'minder-orbit') if g.get('id')=='orbit-art').set('transform','translate(13.5 13.5) scale(.75)')
png('app/android/app/src/main/res/drawable-nodpi/ic_minder_background.png',432,tree=adaptive)
png('app/android/app/src/main/res/drawable-nodpi/ic_minder_foreground.png',432,tree=only('minder-mark'))
(res/'drawable/ic_minder_monochrome.xml').write_text(vector())
for stale in (res/'drawable/ic_minder_foreground.xml',res/'values/minder_icon_colors.xml'):stale.unlink(missing_ok=True)
for version in [26,33]:
 folder=res/f'mipmap-anydpi-v{version}';folder.mkdir(exist_ok=True)
 mono='\n  <monochrome android:drawable="@drawable/ic_minder_monochrome"/>' if version==33 else ''
 (folder/'ic_launcher.xml').write_text('<adaptive-icon xmlns:android="http://schemas.android.com/apk/res/android">\n  <background android:drawable="@drawable/ic_minder_background"/>\n  <foreground android:drawable="@drawable/ic_minder_foreground"/>'+mono+'\n</adaptive-icon>\n')
print('Exported Android adaptive (bitmap layers), monochrome and legacy icons; web icons; favicon; public grant logo.')

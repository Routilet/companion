import {copyFile,mkdir,cp,readFile,writeFile} from 'node:fs/promises';
import {createHash} from 'node:crypto';
const manifest=JSON.parse(await readFile('ui/manifest.json','utf8'));
for(const [name,hash] of Object.entries(manifest.files)){if(createHash('sha256').update(await readFile('ui/'+name)).digest('hex')!==hash)throw Error('UI snapshot mismatch: '+name)}
if(manifest.algorithm!=='sha256'||manifest.version!=='0.1.0')throw Error('Unsupported UI snapshot');
await mkdir('dist',{recursive:true});
await copyFile('style.css','dist/style.css');
await cp('ui','dist/ui',{recursive:true});
const sprite=(await readFile('ui/icons/sprite.svg','utf8')).replace('<svg xmlns="http://www.w3.org/2000/svg">','<svg xmlns="http://www.w3.org/2000/svg" hidden aria-hidden="true">').replace(/<symbol id="([^"]+)"/g,(_,name)=>'<symbol id="routilet-icon-'+name+'"');
const html=await readFile('index.html','utf8');if(!html.includes('<!-- ROUTILET_UI_SPRITE -->'))throw Error('Missing UI sprite marker');
const document=html.replace('<!-- ROUTILET_UI_SPRITE -->',()=>sprite);
const ids=[...document.matchAll(/\bid="([^"]+)"/g)].map(m=>m[1]);if(new Set(ids).size!==ids.length)throw Error('Duplicate HTML or SVG ID');
for(const m of document.matchAll(/<use href="#([^"]+)"/g))if(!ids.includes(m[1]))throw Error('Unresolved icon: '+m[1]);
await writeFile('dist/index.html',document);

await mkdir('dist/icons',{recursive:true});
await copyFile('ui/brand/companion.svg','dist/icons/companion.svg');

await mkdir('dist/vendor',{recursive:true});
for(const [pkg,name] of [['markdown-it','markdown-it.min.js'],['katex','katex.min.js'],['katex','katex.min.css']])await copyFile('node_modules/'+pkg+'/dist/'+name,'dist/vendor/'+name);
await cp('node_modules/katex/dist/fonts','dist/vendor/fonts',{recursive:true});
for(const pkg of ['markdown-it','katex'])await copyFile('node_modules/'+pkg+'/LICENSE','dist/vendor/'+pkg+'-LICENSE.txt');

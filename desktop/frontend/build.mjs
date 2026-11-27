import {copyFile,mkdir,cp} from 'node:fs/promises';
await mkdir('dist',{recursive:true});
for(const name of ['index.html','style.css']) await copyFile(name,'dist/'+name);

await mkdir('dist/icons',{recursive:true});
await copyFile('icons/companion.svg','dist/icons/companion.svg');

await mkdir('dist/vendor',{recursive:true});
for(const [pkg,name] of [['markdown-it','markdown-it.min.js'],['katex','katex.min.js'],['katex','katex.min.css']])await copyFile('node_modules/'+pkg+'/dist/'+name,'dist/vendor/'+name);
await cp('node_modules/katex/dist/fonts','dist/vendor/fonts',{recursive:true});
for(const pkg of ['markdown-it','katex'])await copyFile('node_modules/'+pkg+'/LICENSE','dist/vendor/'+pkg+'-LICENSE.txt');

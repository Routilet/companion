import {copyFile,mkdir} from 'node:fs/promises';
await mkdir('dist',{recursive:true});
for(const name of ['index.html','style.css']) await copyFile(name,'dist/'+name);

await mkdir('dist/icons',{recursive:true});
await copyFile('icons/companion.svg','dist/icons/companion.svg');

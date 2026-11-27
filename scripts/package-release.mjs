import {readFile,writeFile,mkdir,cp,rm} from 'node:fs/promises';
import path from 'node:path';
import {createHash} from 'node:crypto';
import {execFileSync} from 'node:child_process';

const version=(await readFile('VERSION','utf8')).trim();
const pkg=JSON.parse(await readFile('package.json','utf8'));
if(pkg.name!=='@routilet/companion'||pkg.version!==version||!/^0\.(?:[1-9]\d*)\.\d+(?:-(?:beta|rc)\.\d+)?$/.test(version))throw Error('VERSION/package.json mismatch');
const root='out/npm-package';
await rm(root,{recursive:true,force:true});await mkdir(root,{recursive:true});
const manifest={schemaVersion:1,version,platforms:{}};
const targets=[
 ['win32-x64','bin/win32-x64/companion.exe'],
 ['darwin-arm64','bin/darwin-arm64/Companion.app/Contents/MacOS/companion']
];
const notes=[];
for(const [target,relative] of targets){
 const stage=path.join('out','native',target);
 await mkdir(path.join(root,'bin',target),{recursive:true});
 await cp(path.join(stage,'bin'),path.join(root,'bin',target),{recursive:true,preserveTimestamps:true});
 manifest.platforms[target]={executable:relative,sha256:createHash('sha256').update(await readFile(path.join(root,relative))).digest('hex')};
 notes.push('## '+target+'\n'+await readFile(path.join(stage,'GUI-NOTICES.txt'),'utf8')+'\n'+await readFile(path.join(stage,'CLI-NOTICES.txt'),'utf8'));
}
await writeFile(path.join(root,'manifest.json'),JSON.stringify(manifest,null,2)+'\n');
delete pkg.private;pkg.publishConfig={access:'public'};
await writeFile(path.join(root,'package.json'),JSON.stringify(pkg,null,2)+'\n');
await writeFile(path.join(root,'THIRD_PARTY_NOTICES.txt'),notes.join('\n\n'));
for(const n of ['README.md','LICENSE'])await cp(n,path.join(root,n));
await mkdir('release',{recursive:true});
execFileSync('npm',['pack','--ignore-scripts','--pack-destination',path.resolve('release')],{cwd:root,stdio:'inherit'});

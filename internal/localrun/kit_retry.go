package localrun

import ("strings";"time")

// Only retry the documented transient operation-lock conflict. Never retry
// integrity or installation errors.
func retryKitExport(run func()Process,sleep func(time.Duration))Process{
 for attempt:=0;;attempt++{p:=run();if p.State!="failed"||!strings.Contains(p.Stderr,"另一 Routilet 实例正在操作")||attempt>=7{return p};sleep(150*time.Millisecond)}
}

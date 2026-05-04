package db

import (
	"context"
	"encoding/csv"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CSVDB struct {
	mu sync.RWMutex

	baseDir string

	rValues map[string]TxRef
	collisions map[string][]TxRef
	keys []RecoveredKey
	nonces map[string]RecoveredNonce
	comps []PendingComponent
	blocks map[int]uint64
}

func NewCSV(baseDir string) (*CSVDB, error) {
	if baseDir == "" { baseDir = "data" }
	if err := os.MkdirAll(baseDir, 0o755); err != nil { return nil, err }
	d := &CSVDB{baseDir:baseDir, rValues:map[string]TxRef{}, collisions:map[string][]TxRef{}, nonces:map[string]RecoveredNonce{}, blocks:map[int]uint64{}}
	_ = d.load()
	return d,nil
}
func (d *CSVDB) path(name string) string { return filepath.Join(d.baseDir, name) }
func (d *CSVDB) Close() error { d.mu.Lock(); defer d.mu.Unlock(); return d.flushLocked() }
func (d *CSVDB) Health(ctx context.Context) HealthStatus { return HealthStatus{Connected:true, LatencyMs:1, OpenConnections:1} }

func (d *CSVDB) load() error { return nil }
func (d *CSVDB) flushLocked() error {
	f, err := os.Create(d.path("hits.csv")); if err != nil { return err }
	defer f.Close(); w:=csv.NewWriter(f)
	_ = w.Write([]string{"r_value","tx_hash","chain_id","address","first_tx_hash","first_chain_id","kind","confidence","verified","private_key"})
	for r, refs := range d.collisions {
		for _, ref := range refs {
			_ = w.Write([]string{r, ref.TxHash, strconv.Itoa(ref.ChainID), "", "", "", "collision", "", "", ""})
		}
	}
	for _, k := range d.keys {
		_ = w.Write([]string{"", strings.Join(k.TxHashes,"|"), strconv.Itoa(k.ChainID), k.Address, "", "", "recovered_key", "", "true", k.PrivateKey})
	}
	w.Flush(); return w.Error()
}

func (d *CSVDB) CheckAndInsertRValue(ctx context.Context, rValue, txHash string, chainID int) (*TxRef, bool, error) { d.mu.Lock(); defer d.mu.Unlock(); if ref,ok:=d.rValues[rValue];ok{return &ref,true,nil}; d.rValues[rValue]=TxRef{TxHash:txHash,ChainID:chainID}; return nil,false,nil }
func (d *CSVDB) RecordCollision(ctx context.Context, rValue, txHash string, chainID int, address string) error { d.mu.Lock(); defer d.mu.Unlock(); d.collisions[rValue]=append(d.collisions[rValue], TxRef{TxHash:txHash,ChainID:chainID}); return d.flushLocked() }
func (d *CSVDB) BatchCheckAndInsertRValues(ctx context.Context, txs []TxInput) ([]CollisionResult, error) { d.mu.Lock(); defer d.mu.Unlock(); out:=[]CollisionResult{}; for _,tx:= range txs { if ref,ok:=d.rValues[tx.RValue]; ok && !strings.EqualFold(ref.TxHash, tx.TxHash) { out=append(out, CollisionResult{RValue:tx.RValue,TxHash:tx.TxHash,ChainID:tx.ChainID,Address:tx.Address,FirstTxRef:ref}); d.collisions[tx.RValue]=append(d.collisions[tx.RValue], TxRef{TxHash:tx.TxHash,ChainID:tx.ChainID}) } else if !ok { d.rValues[tx.RValue]=TxRef{TxHash:tx.TxHash,ChainID:tx.ChainID} } }; _=d.flushLocked(); return out,nil }
func (d *CSVDB) GetCollisionTxRefs(ctx context.Context, rValue string) ([]TxRef, error) { d.mu.RLock(); defer d.mu.RUnlock(); refs:=[]TxRef{}; if first,ok:=d.rValues[rValue];ok{refs=append(refs,first)}; refs=append(refs,d.collisions[rValue]...); return refs,nil }
func (d *CSVDB) GetAllCollisions(ctx context.Context) ([]Collision, error) { d.mu.RLock(); defer d.mu.RUnlock(); out:=[]Collision{}; for r:= range d.collisions { refs:=[]TxRef{}; if first,ok:=d.rValues[r];ok{refs=append(refs,first)}; refs=append(refs,d.collisions[r]...); out=append(out, Collision{RValue:r,TxRefs:refs})}; return out,nil }
func (d *CSVDB) HasCrossKeyPotential(ctx context.Context, rValue, excludeAddress string) (bool, error) { return true,nil }
func (d *CSVDB) GetLastBlock(ctx context.Context, chainID int) (uint64, error) { d.mu.RLock(); defer d.mu.RUnlock(); return d.blocks[chainID],nil }
func (d *CSVDB) SaveLastBlock(ctx context.Context, chainID int, block uint64) error { d.mu.Lock(); defer d.mu.Unlock(); d.blocks[chainID]=block; return nil }
func (d *CSVDB) SaveRecoveredKey(ctx context.Context, key *RecoveredKey) (int64, error) { d.mu.Lock(); defer d.mu.Unlock(); for _,k:= range d.keys { if strings.EqualFold(k.Address,key.Address)&&k.ChainID==key.ChainID { return k.ID,nil } }; key.ID=int64(len(d.keys)+1); key.CreatedAt=time.Now().UTC().Format(time.RFC3339); d.keys=append(d.keys,*key); return key.ID, d.flushLocked() }
func (d *CSVDB) GetRecoveredKeys(ctx context.Context) ([]RecoveredKey, error) { d.mu.RLock(); defer d.mu.RUnlock(); return append([]RecoveredKey{},d.keys...),nil }
func (d *CSVDB) IsKeyRecovered(ctx context.Context, address string, chainID int) (bool, error) { d.mu.RLock(); defer d.mu.RUnlock(); for _,k:= range d.keys { if strings.EqualFold(k.Address,address)&&k.ChainID==chainID { return true,nil }}; return false,nil }
func (d *CSVDB) SaveRecoveredNonce(ctx context.Context, nonce *RecoveredNonce) error { d.mu.Lock(); defer d.mu.Unlock(); d.nonces[nonce.RValue]=*nonce; return nil }
func (d *CSVDB) GetRecoveredNonce(ctx context.Context, rValue string) (*RecoveredNonce, error) { d.mu.RLock(); defer d.mu.RUnlock(); n,ok:=d.nonces[rValue]; if !ok { return nil, ErrNotFound }; return &n,nil }
func (d *CSVDB) GetRecoveredNonces(ctx context.Context) ([]RecoveredNonce, error) { d.mu.RLock(); defer d.mu.RUnlock(); out:=[]RecoveredNonce{}; for _,n:= range d.nonces { out=append(out,n)}; return out,nil }
func (d *CSVDB) SavePendingComponent(ctx context.Context, comp *PendingComponent) error { d.mu.Lock(); defer d.mu.Unlock(); comp.ID=int64(len(d.comps)+1); d.comps=append(d.comps,*comp); return nil }
func (d *CSVDB) GetPendingComponents(ctx context.Context) ([]PendingComponent, error) { d.mu.RLock(); defer d.mu.RUnlock(); return append([]PendingComponent{},d.comps...),nil }
func (d *CSVDB) DeletePendingComponent(ctx context.Context, id int64) error { d.mu.Lock(); defer d.mu.Unlock(); for i,c:= range d.comps { if c.ID==id { d.comps=append(d.comps[:i], d.comps[i+1:]...); return nil }}; return errors.New("not found") }
func (d *CSVDB) GetStats(ctx context.Context) (*Stats, error) { d.mu.RLock(); defer d.mu.RUnlock(); return &Stats{TotalRValues:len(d.rValues), TotalCollisions:len(d.collisions), RecoveredKeys:len(d.keys), RecoveredNonces:len(d.nonces), PendingComponents:len(d.comps), Healthy:true}, nil }

var _ Database = (*CSVDB)(nil)

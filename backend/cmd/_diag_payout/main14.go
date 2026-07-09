package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db";"caipiao/backend/internal/db/sqlcdb";"caipiao/backend/internal/member";"caipiao/backend/internal/timeutil";"github.com/jackc/pgx/v5/pgtype")
func main(){
  _=godotenv.Load(); ctx:=context.Background(); pool,_:=db.Connect(ctx,config.Load().DatabaseURL,5,1); defer pool.Close()
  q:=sqlcdb.New(pool)
  var mid int64; _=pool.QueryRow(ctx,`SELECT id FROM members WHERE account='vs8888'`).Scan(&mid)
  gid,_:=member.LookupActiveGuajiAccountID(ctx,q,mid)
  tf, tt, _ := timeutil.ParseDateRange("2026-07-01", "2026-07-02")
  rows,_:=q.ListMemberFundRecords(ctx, sqlcdb.ListMemberFundRecordsParams{
    MemberID: mid, GuajiAccountID: gid, TimeFrom: pgtype.Timestamptz{Time:tf,Valid:true}, TimeTo: pgtype.Timestamptz{Time:tt,Valid:true},
    FlowDir: pgtype.Text{String:"income",Valid:true}, RowLimit: 10,
  })
  fmt.Println("income:", len(rows))
  for _, r := range rows { fmt.Printf("  %s scheme=%q delta=%.2f type=%s\n", r.LedgerNo, r.SchemeName, r.DeltaAmount, r.TxnType) }
}

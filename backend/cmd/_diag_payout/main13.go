package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db";"caipiao/backend/internal/db/sqlcdb";"caipiao/backend/internal/member";"github.com/jackc/pgx/v5/pgtype")
func main(){
  _=godotenv.Load(); cfg:=config.Load(); ctx:=context.Background(); pool,_:=db.Connect(ctx,cfg.DatabaseURL,5,1); defer pool.Close()
  q:=sqlcdb.New(pool)
  var mid int64; _=pool.QueryRow(ctx,`SELECT id FROM members WHERE account='vs8888'`).Scan(&mid)
  gid, _:=member.LookupActiveGuajiAccountID(ctx, q, mid)
  rows,_:=q.ListMemberFundRecords(ctx, sqlcdb.ListMemberFundRecordsParams{
    MemberID: mid, GuajiAccountID: gid, FlowDir: pgtype.Text{String:"income",Valid:true},
    RowLimit: 5,
  })
  fmt.Println("income fund records:")
  for _, r := range rows { fmt.Printf("  %s scheme=%q delta=%.2f\n", r.LedgerNo, r.SchemeName, r.DeltaAmount) }
}

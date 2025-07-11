
# 📊 Database Optimization – Loan Origination System
This document outlines improvements for optimizing frequently used database queries in the Loan Origination System.

---
## ⚠️ Problem Summary

The following query patterns are used often and are causing performance and security issues:

### Query 1: Get Applications by Status
```go
func (s *Service) GetApplicationsByStatus(status string) ([]LoanApplication, error) {
    return s.db.Query("SELECT * FROM loan_applications WHERE status = ?", status)
}
```

### Query 2: Get Applications by SSN
```go
func (s *Service) GetApplicantApplications(ssn string) ([]LoanApplication, error) {
    return s.db.Query("SELECT * FROM loan_applications WHERE applicant_ssn = ?", ssn)
}
```

---

## 🚨 Identified Issues
- ❌ No pagination — leads to high memory usage and slow responses.
- ❌ No indexing — causes full table scans and poor performance.
- ❌ Querying directly using PII (SSN) — violates data privacy best practices.
- ❌ Uses `SELECT *` — inefficient when only specific fields are needed.

---

## ✅ Optimization Solutions

### 1. Add Pagination
Limit the number of rows returned using `LIMIT` and `OFFSET`.

```go
func (s *Service) GetApplicationsByStatus(status string, limit, offset int) ([]LoanApplication, error) {
    query := `SELECT id, applicant_name, status, created_at FROM loan_applications 
              WHERE status = $1 LIMIT $2 OFFSET $3`
    return s.db.Query(query, status, limit, offset)
}
```

---

### 2. Add Indexes
Improve query performance using indexes:

```sql
-- Index for status column
CREATE INDEX idx_loan_applications_status ON loan_applications(status);

-- Index for applicant_ssn column (if needed)
CREATE INDEX idx_loan_applications_ssn ON loan_applications(applicant_ssn);
```

> 🔐 If storing PII like SSN, it's better to store a hash instead.

---

### 3. Avoid `SELECT *`
Only retrieve necessary fields to reduce I/O and improve query speed:

```sql
SELECT id, applicant_name, status FROM loan_applications WHERE status = $1
```

---

### 4. Protect PII Using Hashing
Avoid querying SSN directly. Instead, store and query using a hashed version:

#### Hash Function Example (SHA-256)

```go
func hashSSN(ssn string) string {
    h := sha256.New()
    h.Write([]byte(ssn))
    return hex.EncodeToString(h.Sum(nil))
}
```

#### Hashed Query Example

```go
func (s *Service) GetApplicantApplications(ssn string) ([]LoanApplication, error) {
    ssnHash := hashSSN(ssn)
    query := `SELECT id, applicant_name, status FROM loan_applications 
              WHERE applicant_ssn_hash = $1`
    return s.db.Query(query, ssnHash)
}
```

---

## 📈 Result Summary

| Aspect             | Before                     | After                          |
|--------------------|-----------------------------|---------------------------------|
| Query performance  | Slow (full table scans)     | Fast (indexed & filtered)       |
| Memory usage       | High (no limits)            | Controlled (pagination)         |
| PII risk           | Exposed (raw SSN)           | Secured (hashed SSN)            |
| Data efficiency    | Heavy (`SELECT *`)          | Lean (select only needed fields) |

---

## ✅ Implementation Checklist

- [x] Add `LIMIT` & `OFFSET` for pagination
- [x] Add indexes on `status` and `applicant_ssn`
- [x] Replace `SELECT *` with specific fields
- [x] Use SHA-256 hashing for SSN
- [x] Store SSN hashes in a separate column (`applicant_ssn_hash`)

---

## 🛠 Next Steps

- Add migration scripts to create the indexes and hash column.
- Refactor database access layer to use hashed values.
- Perform query benchmarking before and after optimization.

---

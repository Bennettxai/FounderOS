@echo off
REM Pre-demo checklist - Run right before Orgo.ai presentation
REM Usage: scripts\pre_demo_checklist.bat

echo ╔══════════════════════════════════════════════════════════════════╗
echo ║              PRE-DEMO CHECKLIST (Final Verification)            ║
echo ╚══════════════════════════════════════════════════════════════════╝
echo.
echo Run this script 5 minutes before the demo to ensure everything works
echo.

set CHECKS_PASSED=0
set CHECKS_FAILED=0

REM Check 1: Backend server
echo 🔍 [1/10] Checking backend server...
curl -s http://localhost:8001/api/onboarding/status >nul 2>&1
if %ERRORLEVEL% == 0 (
    echo       ✅ Backend responding on port 8001
    set /a CHECKS_PASSED+=1
) else (
    echo       ❌ Backend NOT responding
    echo          Fix: cd desktop\backend-go ^&^& go run cmd\server\main.go
    set /a CHECKS_FAILED+=1
)
echo.

REM Check 2: Frontend server
echo 🔍 [2/10] Checking frontend server...
curl -s http://localhost:5173 >nul 2>&1
if %ERRORLEVEL% == 0 (
    echo       ✅ Frontend responding on port 5173
    set /a CHECKS_PASSED+=1
) else (
    echo       ❌ Frontend NOT responding
    echo          Fix: cd frontend ^&^& npm run dev
    set /a CHECKS_FAILED+=1
)
echo.

REM Check 3: .env file exists
echo 🔍 [3/10] Checking .env file...
if exist .env (
    echo       ✅ .env file found
    set /a CHECKS_PASSED+=1
) else (
    echo       ❌ .env file not found
    echo          Fix: Copy .env.example to .env
    set /a CHECKS_FAILED+=1
)
echo.

REM Check 4: Groq API key
echo 🔍 [4/10] Checking Groq API key...
findstr /C:"GROQ_API_KEY=gsk_" .env >nul 2>&1
if %ERRORLEVEL% == 0 (
    echo       ✅ Groq API key configured
    set /a CHECKS_PASSED+=1
) else (
    echo       ❌ Groq API key missing or invalid
    echo          Fix: Set GROQ_API_KEY in .env
    set /a CHECKS_FAILED+=1
)
echo.

REM Check 5: Google OAuth credentials
echo 🔍 [5/10] Checking Google OAuth...
findstr /C:"GOOGLE_CLIENT_ID=" .env >nul 2>&1
if %ERRORLEVEL% == 0 (
    findstr /C:"GOOGLE_CLIENT_SECRET=" .env >nul 2>&1
    if %ERRORLEVEL% == 0 (
        echo       ✅ Google OAuth credentials configured
        set /a CHECKS_PASSED+=1
    ) else (
        echo       ❌ Google OAuth credentials incomplete
        echo          Fix: Set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET
        set /a CHECKS_FAILED+=1
    )
) else (
    echo       ❌ Google OAuth credentials missing
    echo          Fix: Set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET
    set /a CHECKS_FAILED+=1
)
echo.

REM Check 6: Database connection (using validate script)
echo 🔍 [6/10] Checking database connection...
echo       (Running validate_environment.go for detailed checks)
go run scripts\validate_environment.go >nul 2>&1
if %ERRORLEVEL% == 0 (
    echo       ✅ Database checks passed
    set /a CHECKS_PASSED+=1
) else (
    echo       ⚠️  Some database checks failed (see validate_environment.go)
)
echo.

REM Check 7: Port availability
echo 🔍 [7/10] Checking port availability...
netstat -an | findstr ":8001" >nul 2>&1
if %ERRORLEVEL% == 0 (
    echo       ✅ Port 8001 in use (backend running)
    set /a CHECKS_PASSED+=1
) else (
    echo       ⚠️  Port 8001 not in use (backend may not be running)
)
echo.

REM Check 8: Demo script ready
echo 🔍 [8/10] Checking demo materials...
if exist DEMO_FLOW.md (
    echo       ✅ Demo script ready (DEMO_FLOW.md)
    set /a CHECKS_PASSED+=1
) else (
    echo       ⚠️  Demo script not found
)
echo.

REM Check 9: Scripts directory
echo 🔍 [9/10] Checking diagnostic scripts...
if exist scripts\troubleshooter.go (
    if exist scripts\system_dashboard.go (
        echo       ✅ Diagnostic scripts ready
        set /a CHECKS_PASSED+=1
    ) else (
        echo       ⚠️  Some scripts missing
    )
) else (
    echo       ⚠️  Scripts directory incomplete
)
echo.

REM Check 10: Test data creation capability
echo 🔍 [10/10] Checking test data scripts...
if exist scripts\simulate_onboarding_flow.go (
    echo       ✅ Test data scripts ready
    set /a CHECKS_PASSED+=1
) else (
    echo       ⚠️  Test data scripts not found
)
echo.

REM Final summary
echo ╔══════════════════════════════════════════════════════════════════╗
echo ║                      CHECKLIST SUMMARY                           ║
echo ╚══════════════════════════════════════════════════════════════════╝
echo.
echo    ✅ Passed:  %CHECKS_PASSED%
echo    ❌ Failed:  %CHECKS_FAILED%
echo.

if %CHECKS_FAILED% == 0 (
    echo 🎉 ALL CHECKS PASSED - READY FOR DEMO!
    echo.
    echo 📋 Last-minute tips:
    echo    1. Open browser tabs in advance ^(localhost:5173/onboarding^)
    echo    2. Have DEMO_FLOW.md open in editor
    echo    3. Close unnecessary applications ^(free up RAM^)
    echo    4. Test microphone and screen sharing
    echo    5. Take a deep breath - you got this! 💪
) else (
    echo ❌ SOME CHECKS FAILED - FIX BEFORE DEMO
    echo.
    echo Review the failures above and fix them
)
echo.

pause

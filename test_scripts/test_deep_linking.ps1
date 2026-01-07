# Deep Linking Test Script
# Tests the breadcrumb functionality in the note show endpoint
#
# Usage: .\test_deep_linking.ps1 -Email "your@email.com" -Password "yourpassword"
# Or set environment variables: TEST_EMAIL, TEST_PASSWORD

param(
    [string]$Email = $env:TEST_EMAIL,
    [string]$Password = $env:TEST_PASSWORD,
    [string]$BaseUrl = "http://localhost:3000"
)

# Configuration
$token = "" # Will be set after login

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Deep Linking Breadcrumb Test Script  " -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if credentials are provided
if (-not $Email -or -not $Password) {
    Write-Host "Please provide credentials:" -ForegroundColor Yellow
    Write-Host "  .\test_deep_linking.ps1 -Email 'your@email.com' -Password 'yourpassword'" -ForegroundColor Gray
    Write-Host "  Or set environment variables: TEST_EMAIL, TEST_PASSWORD" -ForegroundColor Gray
    exit 1
}

# Helper function for API calls
function Invoke-Api {
    param(
        [string]$Method,
        [string]$Endpoint,
        [hashtable]$Body = $null,
        [bool]$UseAuth = $true
    )
    
    $headers = @{ "Content-Type" = "application/json" }
    if ($UseAuth -and $script:token) {
        $headers["Authorization"] = "Bearer $script:token"
    }
    
    $uri = "$baseUrl$Endpoint"
    
    try {
        if ($Body) {
            $response = Invoke-RestMethod -Uri $uri -Method $Method -Headers $headers -Body ($Body | ConvertTo-Json -Depth 10)
        }
        else {
            $response = Invoke-RestMethod -Uri $uri -Method $Method -Headers $headers
        }
        return $response
    }
    catch {
        Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
        if ($_.Exception.Response) {
            $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
            Write-Host "Response: $($reader.ReadToEnd())" -ForegroundColor Red
        }
        return $null
    }
}

# Step 1: Login
Write-Host "[1] Logging in as $Email..." -ForegroundColor Yellow
$loginBody = @{
    email    = $Email
    password = $Password
}
$loginResponse = Invoke-Api -Method "POST" -Endpoint "/api/auth/login" -Body $loginBody -UseAuth $false

if ($loginResponse -and $loginResponse.data.access_token) {
    $script:token = $loginResponse.data.access_token
    Write-Host "    Login successful!" -ForegroundColor Green
}
else {
    Write-Host "    Login failed. Please update credentials in script." -ForegroundColor Red
    exit 1
}

# Step 2: Create nested notebook structure
Write-Host ""
Write-Host "[2] Creating nested notebook structure..." -ForegroundColor Yellow

# Create root notebook
$root = Invoke-Api -Method "POST" -Endpoint "/api/notebook/v1" -Body @{ name = "Work" }
$rootId = $root.data.id
Write-Host "    Created: Work (root) - $rootId" -ForegroundColor Gray

# Create child notebook
$child = Invoke-Api -Method "POST" -Endpoint "/api/notebook/v1" -Body @{ name = "Projects"; parent_id = $rootId }
$childId = $child.data.id
Write-Host "    Created: Projects (child of Work) - $childId" -ForegroundColor Gray

# Create grandchild notebook
$grandchild = Invoke-Api -Method "POST" -Endpoint "/api/notebook/v1" -Body @{ name = "Q1 Planning"; parent_id = $childId }
$grandchildId = $grandchild.data.id
Write-Host "    Created: Q1 Planning (child of Projects) - $grandchildId" -ForegroundColor Gray

# Step 3: Create note in the deeply nested notebook
Write-Host ""
Write-Host "[3] Creating note in deeply nested notebook..." -ForegroundColor Yellow
$note = Invoke-Api -Method "POST" -Endpoint "/api/note/v1" -Body @{
    title       = "Meeting Notes"
    content     = "Discussion about Q1 goals"
    notebook_id = $grandchildId
}
$noteId = $note.data.id
Write-Host "    Created note: $noteId" -ForegroundColor Gray

# Step 4: Test deep linking - fetch note with breadcrumb
Write-Host ""
Write-Host "[4] Testing deep link: GET /api/note/v1/$noteId" -ForegroundColor Yellow
$noteDetails = Invoke-Api -Method "GET" -Endpoint "/api/note/v1/$noteId"

if ($noteDetails -and $noteDetails.data) {
    Write-Host ""
    Write-Host "=== DEEP LINK RESPONSE ===" -ForegroundColor Cyan
    Write-Host "Note ID: $($noteDetails.data.id)" -ForegroundColor White
    Write-Host "Title: $($noteDetails.data.title)" -ForegroundColor White
    Write-Host "Notebook ID: $($noteDetails.data.notebook_id)" -ForegroundColor White
    
    Write-Host ""
    Write-Host "Breadcrumb (hierarchy path):" -ForegroundColor Green
    $index = 0
    foreach ($crumb in $noteDetails.data.breadcrumb) {
        $indent = "  " * $index
        Write-Host "$indent-> $($crumb.name) ($($crumb.id))" -ForegroundColor White
        $index++
    }
    
    # Validate breadcrumb
    Write-Host ""
    Write-Host "=== VALIDATION ===" -ForegroundColor Cyan
    $expectedCount = 3
    $actualCount = $noteDetails.data.breadcrumb.Count
    
    if ($actualCount -eq $expectedCount) {
        Write-Host "[PASS] Breadcrumb contains $actualCount items (expected $expectedCount)" -ForegroundColor Green
    }
    else {
        Write-Host "[FAIL] Breadcrumb contains $actualCount items (expected $expectedCount)" -ForegroundColor Red
    }
    
    # Check order (should be root-first)
    if ($noteDetails.data.breadcrumb[0].name -eq "Work" -and 
        $noteDetails.data.breadcrumb[1].name -eq "Projects" -and 
        $noteDetails.data.breadcrumb[2].name -eq "Q1 Planning") {
        Write-Host "[PASS] Breadcrumb order is correct (root-first)" -ForegroundColor Green
    }
    else {
        Write-Host "[FAIL] Breadcrumb order is incorrect" -ForegroundColor Red
    }
    
}
else {
    Write-Host "[FAIL] Could not fetch note details" -ForegroundColor Red
}

# Step 5: Cleanup
Write-Host ""
Write-Host "[5] Cleaning up test data..." -ForegroundColor Yellow
Invoke-Api -Method "DELETE" -Endpoint "/api/note/v1/$noteId" | Out-Null
Invoke-Api -Method "DELETE" -Endpoint "/api/notebook/v1/$grandchildId" | Out-Null
Invoke-Api -Method "DELETE" -Endpoint "/api/notebook/v1/$childId" | Out-Null
Invoke-Api -Method "DELETE" -Endpoint "/api/notebook/v1/$rootId" | Out-Null
Write-Host "    Cleanup complete!" -ForegroundColor Green

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Test Complete!                       " -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

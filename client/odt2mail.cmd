REM Convert Odt to pdf and pass it to report2mail
set lname=%1
set fname=%2
set studyDate=%3
set email=%4
set reportPath=%5
set docName=%~n5
set InstitutionName=MyFacility
<Path to >soffice.exe --headless --convert-to pdf %reportPath%
set SERVERADDR=reports.isis.care:443
report2mail.exe "{\"emailAddress\":\"%email%\",\"firstname\":\"%fname%\",\"lastname\":\"%lname%\",\"date\":\"%studyDate%\",\"office\":\"%InstitutionName%\",\"fileLocation\":\"%docName%.pdf\"}" >> log.txt

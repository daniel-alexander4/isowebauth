Unicode true

!include "MUI2.nsh"
!include "x64.nsh"

!define INFO_PRODUCTNAME "{{.Info.ProductName}}"
!define INFO_COMPANYNAME "{{.Info.CompanyName}}"
!define INFO_PRODUCTVERSION "{{.Info.ProductVersion}}"
!define INFO_COPYRIGHT "{{.Info.Copyright}}"
!define PRODUCT_EXECUTABLE "${INFO_PRODUCTNAME}.exe"
!define UNINST_KEY_NAME "${INFO_COMPANYNAME}${INFO_PRODUCTNAME}"

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\${INFO_PRODUCTNAME}-amd64-installer.exe"
InstallDir "$PROGRAMFILES64\${INFO_PRODUCTNAME}"
RequestExecutionLevel admin

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_ABORTWARNING

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Section
    SetOutPath $INSTDIR
    File "..\..\bin\${PRODUCT_EXECUTABLE}"

    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateDirectory "$SMPROGRAMS\${INFO_PRODUCTNAME}"
    CreateShortCut "$SMPROGRAMS\${INFO_PRODUCTNAME}\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    WriteUninstaller "$INSTDIR\uninstall.exe"

    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY_NAME}" \
        "DisplayName" "${INFO_PRODUCTNAME}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY_NAME}" \
        "UninstallString" '"$INSTDIR\uninstall.exe"'
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY_NAME}" \
        "DisplayIcon" '"$INSTDIR\${PRODUCT_EXECUTABLE}"'
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY_NAME}" \
        "Publisher" "${INFO_COMPANYNAME}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY_NAME}" \
        "DisplayVersion" "${INFO_PRODUCTVERSION}"
SectionEnd

Section "uninstall"
    RMDir /r "$INSTDIR"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"
    RMDir /r "$SMPROGRAMS\${INFO_PRODUCTNAME}"
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY_NAME}"
SectionEnd

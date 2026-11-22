cmake_minimum_required(VERSION 3.21)
project(mcmodlist-gui CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_AUTOMOC ON)
set(CMAKE_AUTORCC ON)

find_package(Qt6 6.5 REQUIRED COMPONENTS Widgets Test Concurrent)

enable_testing()

add_executable(mcmodlist-gui
    src/main.cpp
    src/engine/EngineLocator.cpp
    src/engine/EngineClient.cpp
    src/model/ModTableModel.cpp
    src/model/ViewController.cpp
    src/ui/PaletteUtil.cpp
    src/ui/ChipBar.cpp
    src/ui/DetailsPane.cpp
    src/ui/ModTableView.cpp
    src/ui/ReportDialog.cpp
    src/ui/StartScreen.cpp
    src/ui/MainWindow.cpp
)
target_include_directories(mcmodlist-gui PRIVATE src)
target_link_libraries(mcmodlist-gui PRIVATE Qt6::Widgets Qt6::Concurrent)

# Tests
add_executable(tst_protocol
    tests/protocol_test.cpp
)
target_include_directories(tst_protocol PRIVATE src)
target_compile_definitions(tst_protocol PRIVATE
    MCMODLIST_TESTDATA_DIR="${CMAKE_SOURCE_DIR}/../testdata/protocol")
target_link_libraries(tst_protocol PRIVATE Qt6::Test)
add_test(NAME protocol COMMAND tst_protocol)
set_tests_properties(protocol PROPERTIES ENVIRONMENT "QT_QPA_PLATFORM=offscreen")

add_executable(tst_engineclient
    tests/engineclient_test.cpp
    src/engine/EngineLocator.cpp
    src/engine/EngineClient.cpp
)
target_include_directories(tst_engineclient PRIVATE src)
target_compile_definitions(tst_engineclient PRIVATE
    MCMODLIST_QT_TESTS_DIR="${CMAKE_CURRENT_SOURCE_DIR}/tests")
target_link_libraries(tst_engineclient PRIVATE Qt6::Test Qt6::Concurrent)
add_test(NAME engineclient COMMAND tst_engineclient)
set_tests_properties(engineclient PROPERTIES
    ENVIRONMENT "QT_QPA_PLATFORM=offscreen;MCMODLIST_ENGINE=${CMAKE_BINARY_DIR}/../mcmodlist-engine")

add_executable(tst_mainview
    tests/mainview_test.cpp
    src/engine/EngineLocator.cpp
    src/engine/EngineClient.cpp
    src/model/ModTableModel.cpp
    src/model/ViewController.cpp
    src/ui/PaletteUtil.cpp
    src/ui/ChipBar.cpp
    src/ui/ModTableView.cpp
)
target_include_directories(tst_mainview PRIVATE src)
target_compile_definitions(tst_mainview PRIVATE
    MCMODLIST_QT_TESTS_DIR="${CMAKE_CURRENT_SOURCE_DIR}/tests")
target_link_libraries(tst_mainview PRIVATE Qt6::Widgets Qt6::Test Qt6::Concurrent)
add_test(NAME mainview COMMAND tst_mainview)
set_tests_properties(mainview PROPERTIES
    ENVIRONMENT "QT_QPA_PLATFORM=offscreen;MCMODLIST_ENGINE=${CMAKE_BINARY_DIR}/../mcmodlist-engine")

add_executable(tst_actions
    tests/actions_test.cpp
    src/engine/EngineLocator.cpp
    src/engine/EngineClient.cpp
)
target_include_directories(tst_actions PRIVATE src tests)
target_compile_definitions(tst_actions PRIVATE
    MCMODLIST_QT_TESTS_DIR="${CMAKE_CURRENT_SOURCE_DIR}/tests")
target_link_libraries(tst_actions PRIVATE Qt6::Widgets Qt6::Test Qt6::Concurrent)
add_test(NAME actions COMMAND tst_actions)
set_tests_properties(actions PROPERTIES
    ENVIRONMENT "QT_QPA_PLATFORM=offscreen;MCMODLIST_ENGINE=${CMAKE_BINARY_DIR}/../mcmodlist-engine")

add_executable(tst_polish
    tests/polish_test.cpp
    src/engine/EngineLocator.cpp
    src/engine/EngineClient.cpp
    src/model/ModTableModel.cpp
    src/model/ViewController.cpp
    src/ui/PaletteUtil.cpp
    src/ui/ChipBar.cpp
    src/ui/DetailsPane.cpp
    src/ui/ModTableView.cpp
    src/ui/ReportDialog.cpp
    src/ui/StartScreen.cpp
    src/ui/MainWindow.cpp
)
target_include_directories(tst_polish PRIVATE src tests)
target_link_libraries(tst_polish PRIVATE Qt6::Widgets Qt6::Test Qt6::Concurrent)
add_test(NAME polish COMMAND tst_polish)
set_tests_properties(polish PROPERTIES
    ENVIRONMENT "QT_QPA_PLATFORM=offscreen;MCMODLIST_ENGINE=${CMAKE_BINARY_DIR}/../mcmodlist-engine")

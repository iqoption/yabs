#include <set>
#include <list>
#include <regex>
#include <fstream>
#include <iostream>
#include <cxxabi.h>
#include <unordered_map>

#include <rapidjson/writer.h>
#include <rapidjson/stringbuffer.h>

namespace
{

struct Deleter
{
    void operator()(char *data) const
    {
        free((void *) data);
    }
};

using CharPtr = std::unique_ptr<char, Deleter>;

const std::string ASMJS_RE = "^(?:\\s{4}at\\s){0,1}(?:Array\\.){0,1}([\\w\\d\\$]+)(?: \\(|@).+\\){0,1}";
const std::string WASM_RE = "^.+[\\[\\s](\\d+)[\\]@]\\+{0,1}.+";
}

std::string demangle(const std::string &mangledName)
{
    int status = 0;

    int shift = 0;
    if (mangledName[1] == '_')
    {
        shift = 1;
    }

    CharPtr realname(abi::__cxa_demangle(mangledName.data() + shift, 0, 0, &status));

    if (status == 0)
    {
        return std::string(realname.get());
    } else
    {
        if (mangledName[0] == '_')
        {
            const auto str = mangledName.substr(1, mangledName.size() - 1);
            int status = 0;
            CharPtr realname(abi::__cxa_demangle(str.data(), 0, 0, &status));
            if (status == 0)
            {
                return std::string(realname.get());
            }

            return mangledName;
        }

        return mangledName;
    }
}

void printUsage()
{
    std::cout << "webstackwalker crash_dump symbol_file" << std::endl;
}

std::unordered_map<std::string, std::string> SYMBOL_MAP;

void readSymbols(const std::string &path);

int main(int argc, char **argv)
{
    if (argc < 2)
    {
        printUsage();
        return 1;
    }

    for (int i = 2; i <= (argc - 1); ++i)
    {
        readSymbols(std::string(argv[i]));
    }

    const std::string inputFile(argv[1]);
    std::ifstream input(inputFile);
    const std::regex asmRe(ASMJS_RE);
    const std::regex wasmRe(WASM_RE);

    rapidjson::StringBuffer buffer;
    rapidjson::Writer<rapidjson::StringBuffer> writer(buffer);
    writer.StartArray();

    const std::set<std::string> skip = {"jsStackTrace", "stackTrace", "abort"};

    while (!input.eof())
    {
        std::smatch match;
        std::string line;
        std::getline(input, line);
        std::regex re = asmRe;
        if (line.find("WASM") != std::string::npos || line.find("wasm") != std::string::npos) {
            re = wasmRe;
        }

        if (std::regex_search(line, match, re))
        {
            if (skip.count(match[1]))
            {
                continue;
            }

            auto iter = SYMBOL_MAP.find(match[1]);
            std::string function;
            if (iter != SYMBOL_MAP.cend())
            {
                function = demangle(iter->second);
            } else
            {
                function = demangle(match[1]);
            }

            writer.String(function.c_str());
        }
    }
    writer.EndArray();

    std::cout << buffer.GetString() << std::endl;


    return 0;
}

void readSymbols(const std::string &path)
{
    std::ifstream input(path);

    if (!input.is_open())
    {
        std::cerr << "Can't open symbols file: " << path << std::endl;
        exit(2);
    }

    const std::regex re("^([\\d\\w$]+):([\\d\\w]+)$");

    while (!input.eof())
    {
        std::smatch match;
        std::string line;
        std::getline(input, line);
        if (std::regex_search(line, match, re))
        {
            SYMBOL_MAP[match[1]] = match[2];
        }
    }

    input.close();
}